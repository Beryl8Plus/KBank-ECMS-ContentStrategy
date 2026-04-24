package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/logger"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// Return ContentResult entries that Mass Type
// func cmsPlacementKey(name string) string {
// 	return "cms:placement:" + name
// }

// Return Schedule entries for a placement, used for cache keys and DB queries.
func cmsPlacementSchedulesKey(name string) string {
	return "schedules:placement:" + name + ""
}

// Return logic entries for a placement, used for cache keys.
func ruleDecisionCacheKey(id string) string {
	return "rule:" + id
}

// compile-time interface guard.
var _ DeliveryService = (*CMSDeliveryService)(nil)

type MemoryCache struct {
	Schedules    *cache.CacheMemory[[]*entity.Schedule]
	DecisionRule *cache.CacheMemory[*entity.DecisionRule]
}

func (m *MemoryCache) getStatus() (isMemPressure bool, memoryUsagePct float64) {
	isMemPressureDecision, memoryUsagePctDecision := m.DecisionRule.Status()
	isMemPressureSchedule, memoryUsagePctSchedule := m.Schedules.Status()
	isMemPressure = isMemPressureDecision || isMemPressureSchedule
	memoryUsagePct = math.Max(memoryUsagePctDecision, memoryUsagePctSchedule)
	return isMemPressure, memoryUsagePct
}

// CMSDeliveryService implements DeliveryService.
// Primary path: reads pre-computed content results from Redis.
// Fallback path (cache miss): queries active schedules from PostgreSQL and
// delegates evaluation to cms-runtime via gRPC, then caches the result.
// Background ticker: periodically queries DB, delegates evaluation to gRPC,
// and writes L1/L2/L3 caches for all active placements.
type CMSDeliveryService struct {
	cacheRepo      domainrepo.RedisCacheRepository
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository // nil disables fallback
	decisionRepo   domainrepo.DecisionRuleRepository       // nil disables fallback
	evaluator      RuntimeEvaluator                        // nil disables fallback
	cacheMemory    *MemoryCache                            // nil disables in-memory caching
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
	evaluator RuntimeEvaluator,
	cacheMemory *MemoryCache,
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

// GetCacheKeys returns the list of cache keys for the given placement names.
// This is used for monitoring and debugging purposes to see which cache keys are currently stored in Memory.
func (s *CMSDeliveryService) GetCacheKeys(ctx context.Context) ([]string, error) {
	// For simplicity, this implementation only returns In-memory cache keys.
	var keys []string
	if s.cacheMemory != nil {
		keys = append(keys, s.cacheMemory.Schedules.Keys()...)
		keys = append(keys, s.cacheMemory.DecisionRule.Keys()...)
	}

	return keys, nil
}

// GetCacheStatus returns whether the in-memory cache is under heap pressure
// and the last measured heap utilisation ratio (0–1). Returns false/0 when
// no in-memory cache is configured.
func (s *CMSDeliveryService) GetCacheStatus(ctx context.Context) (isMemPressure bool, memoryUsagePct float64, err error) {
	if s.cacheMemory == nil {
		return false, 0.0, nil
	}
	isMemPressure, memoryUsagePct = s.cacheMemory.getStatus()
	return isMemPressure, memoryUsagePct, nil
}

// FlushCache removes cached results for the given placement names.
// If placementNames is nil or empty, all in-memory entries are cleared and
// ALL Redis placement caches are flushed via FlushDB.
func (s *CMSDeliveryService) FlushCache(ctx context.Context, placementNames []string, isEvaluate bool) error {
	if len(placementNames) == 0 {
		if err := s.cacheRepo.FlushDB(ctx); err != nil {
			return fmt.Errorf("flushing all caches: %w", err)
		}
		return nil
	}

	for _, name := range placementNames {
		placementNameKey := cmsPlacementSchedulesKey(name)
		schedules, ok := s.cacheMemory.Schedules.Get(placementNameKey)
		if ok && schedules != nil {
			for _, sched := range schedules {
				if sched.DecisionRule != nil {
					decisionRuleKey := ruleDecisionCacheKey(sched.DecisionRule.ID.String())
					s.cacheMemory.DecisionRule.Delete(decisionRuleKey)
				}
			}
		}
		s.cacheMemory.Schedules.Delete(placementNameKey)
	}

	// After a full flush, proactively re-populate caches for all active placements to avoid cache stampede on first request.
	// This is optional but helps ensure a warm cache.
	if isEvaluate {
		s.evaluate(ctx)
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
	missedPlacements := make(map[string]struct{})
	for _, placementName := range placementNames {
		result, ok := s.cacheMemory.Schedules.Get(cmsPlacementSchedulesKey(placementName))
		if !ok {
			missedPlacements[placementName] = struct{}{}
			continue
		}
		schedules = append(schedules, result...)
	}

	// Cache miss: re-evaluate to refresh all placement caches, then retry.
	if len(missedPlacements) > 0 && s.occurrenceRepo != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "INFO",
			Message: fmt.Sprintf("schedule cache miss for %d placement(s), triggering evaluate", len(missedPlacements)),
		})
		s.evaluate(ctx)
		for placementName := range missedPlacements {
			if result, ok := s.cacheMemory.Schedules.Get(cmsPlacementSchedulesKey(placementName)); ok {
				schedules = append(schedules, result...)
			}
		}
	}

	// 2. Filter to the requested placement. Re-attach DecisionRule from the rule
	// cache when the schedule was stored lean (UC2 strips DecisionRule on write).
	missedRules := []uuid.UUID{}
	filtered := make(map[string][]*entity.Schedule)
	for _, sched := range schedules {
		if sched.DecisionRule == nil && sched.DecisionRuleID != uuid.Nil {
			decisionRuleKey := ruleDecisionCacheKey(sched.DecisionRuleID.String())
			if rule, ok := s.cacheMemory.DecisionRule.Get(decisionRuleKey); ok && rule != nil {
				sched.DecisionRule = rule
			} else {
				missedRules = append(missedRules, sched.ID)
			}
		}
	}
	// On rule cache miss, fetch from DB and backfill both the schedule pointer and the rule cache.
	if len(missedRules) > 0 {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "INFO",
			Message: fmt.Sprintf("decision rule cache miss for %d schedule(s), querying DB", len(missedRules)),
		})
		if rules, err := s.decisionRepo.GetDecisionRuleByScheduleIDs(ctx, missedRules); err == nil && len(rules) > 0 {
			for _, sched := range schedules {
				if rule, ok := rules[sched.ID]; ok {
					sched.DecisionRule = rule
					s.cacheMemory.DecisionRule.Set(ruleDecisionCacheKey(rule.ID.String()), rule, s.resultTTL)
				}
			}
		} else {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "CMS-DELIVERY",
				Level:   "WARN",
				Message: fmt.Sprintf("Failed to query rules for schedules %v: %v", missedRules, err),
			})
		}
	}
	// 3. Filter to requested placements.
	for _, sched := range schedules {
		if sched.Placement != nil && sched.DecisionRule != nil && slices.Contains(placementNames, sched.Placement.PlacementName) {
			filtered[sched.Placement.PlacementName] = append(filtered[sched.Placement.PlacementName], sched)
		} else {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "CMS-DELIVERY",
				Level:   "WARN",
				Message: fmt.Sprintf("Schedule %v skipped due to missing placement or decision rule", sched.ID),
			})
		}
	}

	// 3.1 Process each placement concurrently via errgroup.
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

// evaluatePlacement handles the full evaluation flow for a single placement name, returning the passing ContentResult entries.
// evaluation for a single placement. Extracted to keep the errgroup callback
// focused and testable independently.
func (s *CMSDeliveryService) evaluatePlacement(
	ctx context.Context,
	cisID, userID, name string,
	schedules []*entity.Schedule,
	resolvedUserAttrs map[string]json.RawMessage,
) []dto.ContentResult {
	var entries []dto.ContentResult
	// Evaluate the placement logic immediately if the evaluator and occurrenceRepo are configured; otherwise, return no results (cache miss).
	if s.evaluator != nil && s.occurrenceRepo != nil {
		results, err := s.evaluator.Evaluate(ctx, name, schedules, resolvedUserAttrs)
		if err != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "CMS-DELIVERY",
				Level:   "WARN",
				Message: fmt.Sprintf("Evaluate failed for %q: %v", name, err),
			})
			return nil
		}
		// filter is LogicEval is true, which means the logic expression passed. The rest of the fields are used for caching and response construction.
		var passing []dto.ContentResult
		for _, item := range results {
			if item.LogicEval {
				passing = append(passing, item)
			}
		}
		entries = passing
	} else {
		return nil
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("Evaluating %d logic entries for placement %q", len(entries), name),
	})
	return entries
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

// runLoop fires evaluate immediately, then on every tick.
func (s *CMSDeliveryService) runLoop(ctx context.Context) {
	defer close(s.done)

	// Fire immediately on start.
	s.evaluate(ctx)

	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evaluate(ctx)
		}
	}
}

// evaluate queries all active schedules from the database,
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
func (s *CMSDeliveryService) evaluate(ctx context.Context) {
	occurrences, err := s.occurrenceRepo.ListActiveAt(ctx, time.Now())
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "ERROR",
			Message: fmt.Sprintf("evaluate: list active occurrences: %v", err),
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
	}

	if len(groups) == 0 {
		return
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("evaluate: processing %d placements (%d schedules)", len(groups), len(schedules)),
	})

	// UC2: cache lean schedules (DecisionRule stripped) for later retrieval in
	// GetPersonalizedContent. Rule data is already stored individually under
	// ruleDecisionCacheKey, so embedding it in every schedule would duplicate it
	// once per schedule that shares the same rule.
	for placementName, g := range groups {
		s.cacheMemory.Schedules.Set(cmsPlacementSchedulesKey(placementName), g.schedules, s.resultTTL)
	}
}

// // setCache writes value to both the in-memory cache (if configured) and Redis.
// func setCache[T any](ctx context.Context, repo domainrepo.RedisCacheRepository, key string, value T, ttl time.Duration) error {
// 	// checking value is string to avoid unnecessary JSON marshal for string values (e.g. rule logic hashes). If value is not string, marshal to JSON before writing to Redis.
// 	if strVal, ok := any(value).(string); ok {
// 		return repo.Set(ctx, key, strVal, ttl)
// 	}
// 	valueJSON, err := json.Marshal(value)
// 	if err != nil {
// 		return fmt.Errorf("cache: marshal error: %w", err)
// 	}
// 	return repo.Set(ctx, key, string(valueJSON), ttl)
// }

// // getCache retrieves a value by key, checking in-memory cache first then falling back to Redis.
// func getCache[T any](ctx context.Context, repo domainrepo.RedisCacheRepository, key string) (T, error) {
// 	var zero T
// 	val, err := repo.Get(ctx, key)
// 	if err != nil {
// 		return zero, err
// 	}
// 	// setCache stores string values raw (without JSON encoding); handle string type directly
// 	// to stay consistent and avoid json.Unmarshal treating "true"/"false" as JSON booleans.
// 	if strPtr, ok := any(&zero).(*string); ok {
// 		*strPtr = val
// 		return zero, nil
// 	}
// 	if err = json.Unmarshal([]byte(val), &zero); err != nil {
// 		return zero, fmt.Errorf("cache: unmarshal error: %w", err)
// 	}
// 	return zero, nil
// }
