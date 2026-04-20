package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	domainservice "kbank-ecms/internal/domain/service"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/util"
)

// cmsPlacementKey returns the Redis key for the given placement name.
func cmsPlacementKey(name string) string {
	return "cms:placement:" + name
}

// cmsPlacementSchedulesKey returns the Redis key where the runtime writes
// ContentResult records for the personalized delivery path.
func cmsPlacementSchedulesKey(name string) string {
	return fmt.Sprintf("cms:placement:%s:schedules", name)
}

// cmsPersonalizedPlacementKey returns the Redis key for per-CIS personalized content.
func cmsPersonalizedPlacementKey(cisID, name string) string {
	return "cms:placement:" + cisID + ":" + name
}

// cmsUserEvalKey returns the Redis key for a per-user logic evaluation result.
func cmsUserEvalKey(userID, hash string) string {
	return "cms:eval:user:" + userID + ":logic:" + hash
}

// ruleDecisionCacheKey returns the Redis key for caching a decision rule by ID.
func ruleDecisionCacheKey(id string) string {
	return "rule:" + id
}

// compile-time interface guard.
var _ domainservice.DeliveryService = (*CMSDeliveryService)(nil)

// CMSDeliveryService implements domainservice.DeliveryService.
// Primary path: reads pre-computed content results from Redis.
// Fallback path (cache miss): queries active schedules from PostgreSQL and
// delegates evaluation to cms-runtime via gRPC, then caches the result.
// Background ticker: periodically queries DB, delegates evaluation to gRPC,
// and writes L1/L2/L3 caches for all active placements.
type CMSDeliveryService struct {
	cacheRepo    domainrepo.RedisCacheRepository
	scheduleRepo domainrepo.ScheduleRepository  // nil disables fallback
	evaluator    domainservice.RuntimeEvaluator // nil disables fallback
	cacheMemory  *cache.CacheMemory[any]
	resultTTL    time.Duration
	tickInterval time.Duration

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	done    chan struct{}
}

// NewCMSDeliveryService creates a CMSDeliveryService.
//   - scheduleRepo may be nil to disable the gRPC fallback.
//   - evaluator may be nil to disable the gRPC fallback.
//   - cacheMemory may be nil to disable local rule caching.
//   - resultTTL is the Redis TTL for results written by the fallback path.
//   - tickInterval is how often the background ticker fires (0 disables ticker).
func NewCMSDeliveryService(
	cacheRepo domainrepo.RedisCacheRepository,
	scheduleRepo domainrepo.ScheduleRepository,
	evaluator domainservice.RuntimeEvaluator,
	cacheMemory *cache.CacheMemory[any],
	resultTTL time.Duration,
	tickInterval time.Duration,
) *CMSDeliveryService {
	return &CMSDeliveryService{
		cacheRepo:    cacheRepo,
		scheduleRepo: scheduleRepo,
		evaluator:    evaluator,
		cacheMemory:  cacheMemory,
		resultTTL:    resultTTL,
		tickInterval: tickInterval,
	}
}

// FlushCache removes cached results for the given placement names.
// If placementNames is nil or empty, all in-memory entries are cleared and
// ALL Redis placement caches are flushed via FlushDB.
func (s *CMSDeliveryService) FlushCache(ctx context.Context, placementNames []string, isEvaluate bool) error {
	if len(placementNames) == 0 {
		if s.cacheMemory != nil {
			s.cacheMemory.Clear()
		}
		if err := s.cacheRepo.FlushDB(ctx); err != nil {
			return fmt.Errorf("flushing all caches: %w", err)
		}

		// After a full flush, proactively re-populate caches for all active placements to avoid cache stampede on first request.
		// This is optional but helps ensure a warm cache.
		if isEvaluate {
			s.evaluateAllViaGRPC(ctx)
		}
		return nil
	}
	for _, name := range placementNames {
		// UC3: flush rule caches for every rule associated with this placement,
		// then flush the schedule list and the placement result cache.
		if schedules, err := getCache[[]*entity.Schedule](ctx, s.cacheMemory, s.cacheRepo, cmsPlacementSchedulesKey(name)); err == nil {
			seen := make(map[string]struct{})
			for _, sched := range schedules {
				id := sched.DecisionRuleID.String()
				if _, already := seen[id]; already {
					continue
				}
				seen[id] = struct{}{}
				_ = s.flushCacheKey(ctx, ruleDecisionCacheKey(id))
			}
		}
		if err := s.flushCacheKey(ctx, cmsPlacementSchedulesKey(name)); err != nil {
			return err
		}
		if err := s.flushCacheKey(ctx, cmsPlacementKey(name)); err != nil {
			return err
		}
	}
	return nil
}

func (s *CMSDeliveryService) flushCacheKey(ctx context.Context, key string) error {
	if s.cacheMemory != nil {
		s.cacheMemory.Delete(key)
	}
	if err := s.cacheRepo.Delete(ctx, key); err != nil {
		return fmt.Errorf("flushing cache key %s: %w", key, err)
	}
	return nil
}

// GetPersonalizedContent evaluates cms:placement:logic:{name} entries against the
// supplied userAttrs, caches per-user evaluation results, and writes a
// personalized cms:placement:{cisID}:{name} cache before returning.
//
// Flow per placement name:
//  1. Check cms:placement:{cisID}:{name} (personalized cache hit) → return immediately.
//  2. Read cms:placement:logic:{name} → []ContentResult.
//  3. For each entry evaluate user attrs (with per-user eval cache).
//  4. Collect passing entries; sort desc by score; write personalized cache; return.
func (s *CMSDeliveryService) GetPersonalizedContent(
	ctx context.Context,
	cisID string,
	userID string,
	placementNames []string,
	userAttrs map[string]json.RawMessage,
) ([]domainservice.ContentResult, error) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("GetPersonalizedContent for cisID %q userID %q placements %v", cisID, userID, placementNames),
	})
	if cisID == "" || userID == "" {
		return nil, fmt.Errorf("GetPersonalizedContent: cisID and userID must not be empty")
	}
	result := make([]domainservice.ContentResult, 0)
	resolvedUserAttrs, resolveErr := s.resolveUserAttrs(ctx, cisID, userAttrs)
	if resolveErr != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "WARN",
			Message: fmt.Sprintf("Failed to resolve user attributes for cisID %q: %v", cisID, resolveErr),
		})
		return nil, resolveErr
	}

	// 1. Query per placement active schedules.
	schedules := []*entity.Schedule{}
	for _, placementName := range placementNames {
		result, err := getCache[[]*entity.Schedule](ctx, s.cacheMemory, s.cacheRepo, cmsPlacementSchedulesKey(placementName))
		if err != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "CMS-DELIVERY",
				Level:   "WARN",
				Message: fmt.Sprintf("Failed to query active schedules for placement %q: %v", placementName, err),
			})
			// On DB failure, skip this placement silently (do not return an error, do not include results).
			continue
		}
		schedules = append(schedules, result...)
	}

	// 2. Filter to the requested placement. Re-attach DecisionRule from the rule
	// cache when the schedule was stored lean (UC2 strips DecisionRule on write).
	filtered := make(map[string][]*entity.Schedule)
	for _, sched := range schedules {
		if sched.DecisionRule == nil {
			if rule, err := getCache[*entity.DecisionRule](ctx, s.cacheMemory, s.cacheRepo, ruleDecisionCacheKey(sched.DecisionRuleID.String())); err == nil {
				sched.DecisionRule = rule
			}
		}
		if sched.Placement != nil && slices.Contains(placementNames, sched.Placement.Name) {
			filtered[sched.Placement.Name] = append(filtered[sched.Placement.Name], sched)
		}
	}

	for _, name := range placementNames {
		personalKey := cmsPersonalizedPlacementKey(cisID, name)
		var entries []domainservice.ContentResult

		// Check the personalized cache first (L3). This should hit when the same user requests the same placement multiple times within resultTTL.
		if cacheEntries, cacheErr := getCache[[]domainservice.ContentResult](
			ctx,
			s.cacheMemory,
			s.cacheRepo,
			personalKey,
		); cacheErr == nil {
			// Cache hit for logic entries.
			entries = cacheEntries
		} else if s.evaluator != nil && s.scheduleRepo != nil {
			// gRPC fallback — one or more rules were missing from cache.
			grpcEntries, grpcErr := s.evaluatePlacementLogicViaGRPC(ctx, name, filtered[name], resolvedUserAttrs)
			if grpcErr != nil {
				logger.LSystem(ctx, entity.SystemLog{
					Service: "CMS-DELIVERY",
					Level:   "WARN",
					Message: fmt.Sprintf("gRPC placement-logic fallback failed for %q: %v", name, grpcErr),
				})
				// On gRPC failure, skip this placement silently (do not return an error, do not include results).
				continue
			}
			entries = grpcEntries
		} else {
			// No logic cache and no gRPC fallback — skip silently.
			continue
		}

		// 3. Evaluate each entry against user attrs.
		var passing []domainservice.ContentResult
		now := time.Now().UTC().Format(time.RFC3339)
		for _, entry := range entries {
			if entry.LogicHash == "" {
				// No logic hash means no user-dependent conditions — treat as match.
				r := entry
				r.LogicEval = true
				r.EvaluatedAt = now
				passing = append(passing, r)
				continue
			}

			// Check per-user eval cache first.
			evalKey := cmsUserEvalKey(userID, entry.LogicHash)
			if cached, cacheErr := getCache[string](ctx, s.cacheMemory, s.cacheRepo, evalKey); cacheErr == nil {
				if cached == "true" {
					r := entry
					r.EvaluatedAt = now
					passing = append(passing, r)
				}
				continue
			}

			// Cache miss — evaluate live.
			cacheVal := "false"
			if entry.LogicEval {
				cacheVal = "true"
				r := entry
				r.EvaluatedAt = now
				passing = append(passing, r)
			}
			_ = setCache(ctx, s.cacheMemory, s.cacheRepo, evalKey, cacheVal, s.resultTTL)
		}

		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "INFO",
			Message: fmt.Sprintf("Evaluating %d logic entries for placement %q", len(entries), name),
		})
		if len(passing) > 0 {
			_ = setCache(ctx, s.cacheMemory, s.cacheRepo, personalKey, passing, s.resultTTL)
		}

		result = append(result, passing...)
	}

	return result, nil
}

// evaluatePlacementLogicViaGRPC queries active schedules, delegates ContentResult
// evaluation to cms-runtime via gRPC, caches the result under cmsPlacementLogicKey, and
// returns the entries for immediate use by GetPersonalizedContent.
func (s *CMSDeliveryService) evaluatePlacementLogicViaGRPC(
	ctx context.Context,
	placementName string,
	filtered []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) ([]domainservice.ContentResult, error) {
	// Delegate evaluation to cms-runtime via gRPC.
	entries, err := s.evaluator.Evaluate(ctx, placementName, filtered, userAttrs)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// ---------------------------------------------------------------------------
// Background ticker — periodically evaluates all active placements via gRPC
// and populates L1 (CacheMemory), L2 (rule-logic hashes), L3 (placement-logic).
// ---------------------------------------------------------------------------

// Start launches the background evaluation ticker.
// It is safe to call multiple times; only the first call starts the loop.
func (s *CMSDeliveryService) Start(ctx context.Context) error {
	if s.tickInterval <= 0 {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "INFO",
			Message: "Background ticker disabled (tickInterval <= 0)",
		})
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.done = make(chan struct{})

	go s.runLoop(ctx)

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("Background ticker started (interval=%s)", s.tickInterval),
	})
	return nil
}

// Stop signals the background loop to exit and waits for it to finish.
func (s *CMSDeliveryService) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	<-s.done
	logger.LSystem(context.Background(), entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: "Background ticker stopped",
	})
	return nil
}

// runLoop fires evaluateAllViaGRPC immediately, then on every tick.
func (s *CMSDeliveryService) runLoop(ctx context.Context) {
	defer close(s.done)

	// Fire immediately on start.
	s.evaluateAllViaGRPC(ctx)

	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evaluateAllViaGRPC(ctx)
		}
	}
}

// evaluateAllViaGRPC queries all active schedules from the database,
// groups them by placement, and for each placement:
//
//  1. L1 — caches each schedule's rules in CacheMemory.
//  2. Calls gRPC Evaluate to obtain ContentResult entries
//     (each entry includes a LogicHash and Conditions).
//  3. L2 — writes each entry's JSON to Redis under cms:rule_logic:v1:{hash}.
//  4. L3 — writes the full entries slice to Redis under cms:placement:logic:{name}.
//
// On gRPC failure for a single placement the error is logged and the loop
// continues with the next placement.
func (s *CMSDeliveryService) evaluateAllViaGRPC(ctx context.Context) {
	schedules, err := util.GetSet(ctx, s.cacheRepo, "schedule:all-active", s.resultTTL, func(ctx context.Context) ([]*entity.Schedule, error) {
		return s.scheduleRepo.ListActiveSchedulesInWindow(ctx, time.Now())
	})
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "ERROR",
			Message: fmt.Sprintf("evaluateAllViaGRPC: list active schedules: %v", err),
		})
		return
	}

	// Group schedules by placement name; derive maxResults per placement.
	type placementGroup struct {
		schedules []*entity.Schedule
	}
	groups := make(map[string]*placementGroup)
	for _, sched := range schedules {
		if sched.Placement == nil || sched.Placement.Name == "" {
			continue
		}
		name := sched.Placement.Name
		g, ok := groups[name]
		if !ok {
			g = &placementGroup{}
			groups[name] = g
		}
		g.schedules = append(g.schedules, sched)

		// L1 — cache individual decision rule in CacheMemory for local look-ups.
		if s.cacheMemory != nil && sched.DecisionRule != nil {
			_ = setCache(ctx, s.cacheMemory, s.cacheRepo, ruleDecisionCacheKey(sched.DecisionRule.ID.String()), sched.DecisionRule, s.resultTTL)
		}
	}

	if len(groups) == 0 {
		return
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("evaluateAllViaGRPC: processing %d placements (%d schedules)", len(groups), len(schedules)),
	})

	// UC2: cache lean schedules (DecisionRule stripped) for later retrieval in
	// GetPersonalizedContent. Rule data is already stored individually under
	// ruleDecisionCacheKey, so embedding it in every schedule would duplicate it
	// once per schedule that shares the same rule.
	for placementName, g := range groups {
		leanSchedules := make([]*entity.Schedule, len(g.schedules))
		for i, sched := range g.schedules {
			lean := *sched
			lean.DecisionRule = nil
			leanSchedules[i] = &lean
		}
		_ = setCache(ctx, s.cacheMemory, s.cacheRepo, cmsPlacementSchedulesKey(placementName), leanSchedules, s.resultTTL)
	}
}

// setCache writes value to both the in-memory cache (if configured) and Redis.
func setCache[T any](ctx context.Context, mem *cache.CacheMemory[any], repo domainrepo.RedisCacheRepository, key string, value T, ttl time.Duration) error {
	if mem != nil {
		mem.Set(key, value, ttl)
	}
	// checking value is string to avoid unnecessary JSON marshal for string values (e.g. rule logic hashes). If value is not string, marshal to JSON before writing to Redis.
	if strVal, ok := any(value).(string); ok {
		return repo.Set(ctx, key, strVal, ttl)
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal error: %w", err)
	}
	return repo.Set(ctx, key, string(valueJSON), ttl)
}

// getCache retrieves a value by key, checking in-memory cache first then falling back to Redis.
func getCache[T any](ctx context.Context, mem *cache.CacheMemory[any], repo domainrepo.RedisCacheRepository, key string) (T, error) {
	var zero T
	if mem != nil {
		if val, ok := mem.Get(ctx, key); ok && val != nil {
			// checking value is string to avoid unnecessary JSON unmarshal for string values (e.g. rule logic hashes).
			// If value is not string, it means it's stored as JSON in cache, so we need to marshal it back to original type before returning.
			if strVal, isStr := any(val).(string); isStr {
				if err := json.Unmarshal([]byte(strVal), &zero); err != nil {
					return zero, fmt.Errorf("cache: unmarshal error: %w", err)
				}
				return zero, nil
			}
			return val.(T), nil
		}
	}
	val, err := repo.Get(ctx, key)
	if err != nil {
		return zero, err
	}
	// setCache stores string values raw (without JSON encoding); handle string type directly
	// to stay consistent and avoid json.Unmarshal treating "true"/"false" as JSON booleans.
	if strPtr, ok := any(&zero).(*string); ok {
		*strPtr = val
		return zero, nil
	}
	if err = json.Unmarshal([]byte(val), &zero); err != nil {
		return zero, fmt.Errorf("cache: unmarshal error: %w", err)
	}
	return zero, nil
}
