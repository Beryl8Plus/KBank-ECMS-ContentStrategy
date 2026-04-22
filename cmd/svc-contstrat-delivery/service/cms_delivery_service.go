package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	domainservice "kbank-ecms/internal/domain/service"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/util"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
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
	cacheRepo      domainrepo.RedisCacheRepository
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository // nil disables fallback
	decisionRepo   domainrepo.DecisionRuleRepository       // nil disables fallback
	evaluator      domainservice.RuntimeEvaluator          // nil disables fallback
	cacheMemory    *cache.CacheMemory[any]
	resultTTL      time.Duration
	tickInterval   time.Duration

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	done    chan struct{}
}

// NewCMSDeliveryService creates a CMSDeliveryService.
//   - occurrenceRepo may be nil to disable the gRPC fallback.
//   - evaluator may be nil to disable the gRPC fallback.
//   - cacheMemory may be nil to disable local rule caching.
//   - resultTTL is the Redis TTL for results written by the fallback path.
//   - tickInterval is how often the background ticker fires (0 disables ticker).
func NewCMSDeliveryService(
	cacheRepo domainrepo.RedisCacheRepository,
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository,
	decisionRuleRepo domainrepo.DecisionRuleRepository,
	evaluator domainservice.RuntimeEvaluator,
	cacheMemory *cache.CacheMemory[any],
	resultTTL time.Duration,
	tickInterval time.Duration,
) *CMSDeliveryService {
	return &CMSDeliveryService{
		cacheRepo:      cacheRepo,
		occurrenceRepo: occurrenceRepo,
		decisionRepo:   decisionRuleRepo,
		evaluator:      evaluator,
		cacheMemory:    cacheMemory,
		resultTTL:      resultTTL,
		tickInterval:   tickInterval,
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
) ([]dto.ContentResult, error) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("GetPersonalizedContent for cisID %q userID %q placements %v", cisID, userID, placementNames),
	})
	if cisID == "" || userID == "" {
		return nil, fmt.Errorf("GetPersonalizedContent: cisID and userID must not be empty")
	}
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
		if sched.DecisionRule == nil && sched.DecisionRuleID != uuid.Nil {
			decisionRuleKey := ruleDecisionCacheKey(sched.DecisionRuleID.String())
			if rule, err := getCache[*entity.DecisionRule](ctx, s.cacheMemory, s.cacheRepo, decisionRuleKey); err == nil && rule != nil {
				sched.DecisionRule = rule
			} else {
				// miss cache, try to query DB directly to avoid stale cache.
				if rule, err := util.GetSet(ctx, s.cacheRepo, decisionRuleKey, s.resultTTL, func(ctx context.Context) (*entity.DecisionRule, error) {
					return s.decisionRepo.GetDecisionRuleByScheduleID(ctx, sched.ID)
				}); err == nil && rule != nil {
					sched.DecisionRule = rule
				} else {
					logger.LSystem(ctx, entity.SystemLog{
						Service: "CMS-DELIVERY",
						Level:   "WARN",
						Message: fmt.Sprintf("Failed to query rule for sched %q: %v", sched.DecisionRuleID.String(), err),
					})
				}
			}
		}
		if sched.Placement != nil && slices.Contains(placementNames, sched.Placement.PlacementName) {
			filtered[sched.Placement.PlacementName] = append(filtered[sched.Placement.PlacementName], sched)
		}
	}

	// 3. Process each placement concurrently via errgroup.
	// Bounded concurrency avoids overwhelming the gRPC backend during
	// cache-miss storms while still providing significant speedup.
	// Each goroutine writes to its own slot (no mutex needed), preserving
	// placementNames order in the merged result.
	const maxPlacementConcurrency = 10
	results := make([][]dto.ContentResult, len(placementNames))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxPlacementConcurrency)

	for i, name := range placementNames {
		g.Go(func() error {
			passing := s.evaluatePlacement(gctx, cisID, userID, name, filtered[name], resolvedUserAttrs)
			if len(passing) > 0 {
				results[i] = passing
			}
			return nil // individual placement failures are non-fatal
		})
	}

	// Wait for all goroutines; errors are swallowed (non-fatal per spec).
	_ = g.Wait()

	var result []dto.ContentResult
	for _, r := range results {
		result = append(result, r...)
	}
	return result, nil
}

// evaluatePlacement handles cache lookup, gRPC fallback, and per-entry
// evaluation for a single placement. Extracted to keep the errgroup callback
// focused and testable independently.
func (s *CMSDeliveryService) evaluatePlacement(
	ctx context.Context,
	cisID, userID, name string,
	schedules []*entity.Schedule,
	resolvedUserAttrs map[string]json.RawMessage,
) []dto.ContentResult {
	personalKey := cmsPersonalizedPlacementKey(cisID, name)
	var entries []dto.ContentResult

	// Check the personalized cache first (L3).
	if cacheEntries, cacheErr := getCache[[]dto.ContentResult](
		ctx,
		s.cacheMemory,
		s.cacheRepo,
		personalKey,
	); cacheErr == nil {
		entries = cacheEntries
	} else if s.evaluator != nil && s.occurrenceRepo != nil {
		// gRPC fallback — one or more rules were missing from cache.
		grpcEntries, grpcErr := s.evaluatePlacementLogicViaGRPC(ctx, name, schedules, resolvedUserAttrs)
		if grpcErr != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "CMS-DELIVERY",
				Level:   "WARN",
				Message: fmt.Sprintf("gRPC placement-logic fallback failed for %q: %v", name, grpcErr),
			})
			return nil
		}
		entries = grpcEntries
	} else {
		return nil
	}

	// 4. Evaluate each entry against user attrs.
	var passing []dto.ContentResult
	now := time.Now().UTC().Format(time.RFC3339)
	for _, entry := range entries {
		if entry.LogicHash == "" {
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

	return passing
}

// evaluatePlacementLogicViaGRPC queries active schedules, delegates ContentResult
// evaluation to cms-runtime via gRPC, caches the result under cmsPlacementLogicKey, and
// returns the entries for immediate use by GetPersonalizedContent.
func (s *CMSDeliveryService) evaluatePlacementLogicViaGRPC(
	ctx context.Context,
	placementName string,
	filtered []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) ([]dto.ContentResult, error) {
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
	occurrences, err := s.occurrenceRepo.ListActiveAt(ctx, time.Now())
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "ERROR",
			Message: fmt.Sprintf("evaluateAllViaGRPC: list active occurrences: %v", err),
		})
		return
	}

	// Deduplicate schedules by ScheduleID (a schedule may have multiple active occurrences).
	seen := make(map[uuid.UUID]struct{})
	schedules := make([]*entity.Schedule, 0, len(occurrences))
	for _, occ := range occurrences {
		if occ.Schedule == nil {
			continue
		}
		if _, dup := seen[occ.ScheduleID]; dup {
			continue
		}
		seen[occ.ScheduleID] = struct{}{}
		schedules = append(schedules, occ.Schedule)
	}

	// Group schedules by placement name; derive maxResults per placement.
	type placementGroup struct {
		schedules []*entity.Schedule
	}
	groups := make(map[string]*placementGroup)
	for _, sched := range schedules {
		if sched.Placement == nil || sched.Placement.PlacementName == "" {
			continue
		}
		name := sched.Placement.PlacementName
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
		_ = setCache(ctx, s.cacheMemory, s.cacheRepo, cmsPlacementSchedulesKey(placementName), g.schedules, s.resultTTL)
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
