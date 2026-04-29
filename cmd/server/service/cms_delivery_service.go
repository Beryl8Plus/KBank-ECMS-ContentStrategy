package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"time"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/infrastructure/pubsub"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

func computePlacementHash(schedules []*entity.Schedule) string {
	h := sha256.New()
	for _, s := range schedules {
		h.Write([]byte(s.ID.String()))
		h.Write([]byte(s.UpdatedAt.String()))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Return Schedule entries for a placement, used for cache keys and DB queries.
func cmsPlacementSchedulesKey(name string) string {
	return "schedules:placement:" + name
}

// Return logic entries for a placement, used for cache keys.
func ruleDecisionCacheKey(id string) string {
	return "rule:" + id
}

// compile-time interface guard.
var _ DeliveryService = (*CMSDeliveryService)(nil)

// SyncPingMessage is re-exported from the shared pubsub package so existing
// call sites in this package keep compiling while the message contract lives
// in one place.
type SyncPingMessage = pubsub.SyncPingMessage

// MemoryCache represents the L1 in-memory mirror.
type MemoryCache struct {
	Schedules     *cache.CacheMemory[[]*entity.Schedule]
	DecisionRule  *cache.CacheMemory[*entity.DecisionRule]
	VersionHashes *cache.CacheMemory[string]    // Map of placement name -> version hash
	LastSync      *cache.CacheMemory[time.Time] // Map of placement name -> last sync timestamp
}

// UpdateSchedules replaces the entire schedule and rule sets in the cache,
// deleting any keys that are not present in the new sets to maintain mirror fidelity.
func (m *MemoryCache) UpdateSchedules(
	newSchedules map[string][]*entity.Schedule,
	newRules map[string]*entity.DecisionRule,
	newVersions map[string]string,
	ttl time.Duration,
	onUpdate func(placement, version string),
) {
	// 1. Sync Schedules
	existingSchedules := m.Schedules.Keys()
	newScheduleKeys := make(map[string]struct{}, len(newSchedules))
	for k, v := range newSchedules {
		m.Schedules.Set(k, v, ttl)
		newScheduleKeys[k] = struct{}{}
	}
	for _, k := range existingSchedules {
		if _, ok := newScheduleKeys[k]; !ok {
			m.Schedules.Delete(k)
		}
	}

	// 2. Sync DecisionRules
	existingRules := m.DecisionRule.Keys()
	newRuleKeys := make(map[string]struct{}, len(newRules))
	for k, v := range newRules {
		m.DecisionRule.Set(k, v, ttl)
		newRuleKeys[k] = struct{}{}
	}
	for _, k := range existingRules {
		if _, ok := newRuleKeys[k]; !ok {
			m.DecisionRule.Delete(k)
		}
	}

	// 3. Sync VersionHashes
	if m.VersionHashes != nil {
		existingVersions := m.VersionHashes.Keys()
		newVersionKeys := make(map[string]struct{}, len(newVersions))
		for k, v := range newVersions {
			m.VersionHashes.Set(k, v, ttl)
			newVersionKeys[k] = struct{}{}

			if onUpdate != nil {
				onUpdate(k, v)
			}
		}
		for _, k := range existingVersions {
			if _, ok := newVersionKeys[k]; !ok {
				m.VersionHashes.Delete(k)
			}
		}
	}

	// 4. Sync LastSync timestamps
	if m.LastSync != nil {
		now := time.Now()
		existingSyncs := m.LastSync.Keys()
		newSyncKeys := make(map[string]struct{}, len(newSchedules))
		for k := range newSchedules {
			// Extract placement name from key (schedules:placement:NAME)
			name := strings.TrimPrefix(k, "schedules:placement:")
			m.LastSync.Set(name, now, ttl)
			newSyncKeys[name] = struct{}{}
		}
		for _, k := range existingSyncs {
			if _, ok := newSyncKeys[k]; !ok {
				m.LastSync.Delete(k)
			}
		}
	}
}

// PruneOrphanedRules removes rules from the cache that are no longer referenced by any schedule.
func (m *MemoryCache) PruneOrphanedRules() {
	if m.Schedules == nil || m.DecisionRule == nil {
		return
	}

	// 1. Collect all Rule IDs currently referenced by ANY cached schedule
	usedRuleKeys := make(map[string]struct{})
	for _, key := range m.Schedules.Keys() {
		if item, ok := m.Schedules.Get(key); ok {
			for _, s := range item {
				if s.DecisionRuleID != uuid.Nil {
					usedRuleKeys[ruleDecisionCacheKey(s.DecisionRuleID.String())] = struct{}{}
				}
			}
		}
	}

	// 2. Delete any cached rules that are no longer referenced
	for _, ruleKey := range m.DecisionRule.Keys() {
		if _, used := usedRuleKeys[ruleKey]; !used {
			m.DecisionRule.Delete(ruleKey)
		}
	}
}

func (m *MemoryCache) getStatus() (isMemPressure bool, memoryUsagePct float64) {
	isMemPressureDecision, memoryUsagePctDecision := m.DecisionRule.Status()
	isMemPressureSchedule, memoryUsagePctSchedule := m.Schedules.Status()
	// Avg the memory usage percentage across both caches to get an overall view of memory pressure,
	// since both caches contribute to total memory usage. This is a simplification;
	// in a real implementation we might want to weight them differently or track total memory usage directly.
	isMemPressure = isMemPressureDecision || isMemPressureSchedule
	memoryUsagePct = (memoryUsagePctDecision + memoryUsagePctSchedule) / 2
	return isMemPressure, memoryUsagePct
}

// CMSDeliveryService implements DeliveryService.
// Primary path: serves content from the in-memory mirror populated by the background ticker.
// Fallback path (cache miss): queries active schedules from PostgreSQL, evaluates rules
// in-process via LocalEvaluator, and populates the mirror.
// Background ticker: periodically queries DB and refreshes the L1/L2 in-memory caches for all active placements.
type CMSDeliveryService struct {
	cacheRepo      domainrepo.RedisCacheRepository
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository // nil disables fallback
	decisionRepo   domainrepo.DecisionRuleRepository       // nil disables fallback
	evaluator      RuntimeEvaluator                        // nil disables fallback
	cacheMemory    *MemoryCache                            // nil disables in-memory caching
	resultTTL      time.Duration
	tickInterval   time.Duration
	leadRepo       domainrepo.LeadRepository // nil disables SALES_TARGET lead enrichment

	customerProfileRepo   domainrepo.CustomerProfileRepository // nil disables CLEN customer-profile integration
	customerProfileEnrich CustomerProfileEnrichConfig          // CacheTTL for the enriched user-attrs blob

	schemaRegistryRepo domainrepo.CLENSchemaRegistryRepository // nil disables schema-driven CLEN enrichment
	schemaFieldsCache  sync.Map                                // map[uuid.UUID]map[string]struct{} — parsed SchemaDefinition fields per registry ID

	attributeRepo            domainrepo.AttributeRepository // nil disables UUID-key transform on resolveUserAttrs return
	attrFieldToUUIDByDsCache sync.Map                       // map[string]map[string]uuid.UUID — field-name → attribute-UUID per datasource

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	done    chan struct{}

	// Subscriber fields
	subCtx    context.Context
	subCancel context.CancelFunc
	subDone   chan struct{}

	// Metrics
	mPlacementVersion *prometheus.GaugeVec
}

// NewCMSDeliveryService creates a CMSDeliveryService.
//   - occurrenceRepo may be nil to disable the DB fallback.
//   - evaluator may be nil to disable in-process rule evaluation.
//   - cacheMemory may be nil to disable local rule caching.
//   - resultTTL is the Redis TTL for results written by the fallback path.
//   - tickInterval is how often the background ticker fires (0 disables ticker).
//   - leadRepo may be nil to disable SALES_TARGET lead enrichment.
//   - customerProfileRepo may be nil to disable CLEN Customer Profile integration.
//   - customerProfileEnrich.CacheTTL controls the Redis TTL for the enriched user-attrs blob.
func NewCMSDeliveryService(
	cacheRepo domainrepo.RedisCacheRepository,
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository,
	decisionRuleRepo domainrepo.DecisionRuleRepository,
	evaluator RuntimeEvaluator,
	cacheMemory *MemoryCache,
	resultTTL time.Duration,
	tickInterval time.Duration,
	leadRepo domainrepo.LeadRepository,
	customerProfileRepo domainrepo.CustomerProfileRepository,
	customerProfileEnrich CustomerProfileEnrichConfig,
	schemaRegistryRepo domainrepo.CLENSchemaRegistryRepository,
	attributeRepo domainrepo.AttributeRepository,
) *CMSDeliveryService {
	// Initialize metrics
	placementVersion := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cms_delivery_placement_version",
		Help: "Indicates the current version hash of a mirrored placement on this pod (Value is always 1)",
	}, []string{"placement_name", "version_hash"})

	// Register with collision handling (safe for parallel tests)
	if err := prometheus.Register(placementVersion); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			if existing, ok := are.ExistingCollector.(*prometheus.GaugeVec); ok {
				placementVersion = existing
			}
		} else {
			// In production, register should only fail on fatal misconfiguration.
			// In tests, we might want to avoid panic.
			logger.LSystem(context.Background(), entity.SystemLog{
				Service: "CMS-DELIVERY",
				Message: fmt.Sprintf("Prometheus metric registration failed: %v", err),
			})
		}
	}

	return &CMSDeliveryService{
		cacheRepo:             cacheRepo,
		occurrenceRepo:        occurrenceRepo,
		decisionRepo:          decisionRuleRepo,
		evaluator:             evaluator,
		cacheMemory:           cacheMemory,
		resultTTL:             resultTTL,
		tickInterval:          tickInterval,
		leadRepo:              leadRepo,
		customerProfileRepo:   customerProfileRepo,
		customerProfileEnrich: customerProfileEnrich,
		schemaRegistryRepo:    schemaRegistryRepo,
		attributeRepo:         attributeRepo,
		done:                  make(chan struct{}),
		mPlacementVersion:     placementVersion,
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

// GetCacheValue returns the cached value for the given key. This is used for monitoring and debugging purposes to inspect the contents of specific cache entries.
func (s *CMSDeliveryService) GetCacheValue(ctx context.Context, key string) (json.RawMessage, error) {
	// For simplicity, this implementation only checks In-memory cache.
	if s.cacheMemory != nil {
		prefixes := []string{"schedules:", "rule:"}
		if !slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(key, p) }) {
			return nil, fmt.Errorf("GetCacheValue: unsupported key prefix for key %q", key)
		}
		// for key "schedules:", return the cached []*entity.Schedule as JSON; for key "rule:{id}", return the cached *entity.DecisionRule as JSON.
		if strings.HasPrefix(key, "schedules:") {
			if val, ok := s.cacheMemory.Schedules.Get(key); ok && val != nil {
				valJSON, err := json.Marshal(val)
				if err != nil {
					return nil, fmt.Errorf("GetCacheValue: marshal error for key %q: %w", key, err)
				}
				return valJSON, nil
			}
		} else if strings.HasPrefix(key, "rule:") {
			if val, ok := s.cacheMemory.DecisionRule.Get(key); ok && val != nil {
				valJSON, err := json.Marshal(val)
				if err != nil {
					return nil, fmt.Errorf("GetCacheValue: marshal error for key %q: %w", key, err)
				}
				return valJSON, nil
			}
		}
	}
	return nil, fmt.Errorf("GetCacheValue: key %q not found in in-memory cache", key)
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
		// Full flush: wipe Redis and in-memory caches entirely.
		if err := s.cacheRepo.FlushDB(ctx); err != nil {
			return fmt.Errorf("flushing all caches: %w", err)
		}
		if s.cacheMemory != nil {
			s.cacheMemory.Schedules.Clear()
			s.cacheMemory.DecisionRule.Clear()
		}
	} else if s.cacheMemory != nil {
		// Selective flush: evict only the named placements from the in-memory mirror.
		// Redis is left intact; the next evaluate() tick will re-populate it.
		for _, name := range placementNames {
			placementNameKey := cmsPlacementSchedulesKey(name)
			schedules, ok := s.cacheMemory.Schedules.Get(placementNameKey)
			if ok && schedules != nil {
				for _, sched := range schedules {
					if sched.DecisionRule != nil {
						s.cacheMemory.DecisionRule.Delete(ruleDecisionCacheKey(sched.DecisionRule.ID.String()))
					}
				}
			}
			s.cacheMemory.Schedules.Delete(placementNameKey)
		}
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
	customerInfo *dto.CustomerRequest,
	channel string,
	placementNames []string,
) ([]dto.ContentResult, error) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("GetPersonalizedContent for placements %v", placementNames),
	})
	if customerInfo == nil || customerInfo.IsEmpty() {
		return nil, fmt.Errorf("GetPersonalizedContent: customerId and customerType must not be empty")
	}

	// 1. Query per placement active schedules.
	schedules := []*entity.Schedule{}
	missedPlacements := make(map[string]struct{})
	for _, placementName := range placementNames {
		// Priority 3: Strict Integrity Fail-Fast
		// If LastSync is too old, the mirror is untrusted.
		if s.cacheMemory != nil && s.cacheMemory.LastSync != nil {
			if lastSync, ok := s.cacheMemory.LastSync.Get(placementName); ok {
				// Default to 10m staleness if tickInterval is not set; otherwise 2x interval.
				threshold := 10 * time.Minute
				if s.tickInterval > 0 {
					threshold = 2 * s.tickInterval
				}
				if time.Since(lastSync) > threshold {
					logger.LSystem(ctx, entity.SystemLog{
						Service: "CMS-DELIVERY",
						Level:   "ERROR",
						Message: fmt.Sprintf("STALE MIRROR DETECTED: placement %q last synced at %s (threshold %s)", placementName, lastSync.Format(time.RFC3339), threshold),
					})
					// Architecture #3: Lazy Self-Heal
					// Attempt a synchronous evaluate once before failing.
					s.evaluate(ctx)
					// Re-check after evaluate
					if refreshedSync, ok := s.cacheMemory.LastSync.Get(placementName); ok && time.Since(refreshedSync) <= threshold {
						logger.LSystem(ctx, entity.SystemLog{
							Service: "CMS-DELIVERY",
							Level:   "INFO",
							Message: fmt.Sprintf("Mirror for %q self-healed successfully", placementName),
						})
					} else {
						// Architecture #5: Strict Integrity Fail-Fast
						return nil, fmt.Errorf("data integrity error: mirror for placement %q is stale and cannot be verified", placementName)
					}
				}
			}
		}

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
		// Evaluate synchronously here to ensure caches are populated before the retry.
		// This adds latency to the request but ensures a better experience for subsequent requests,
		// which is critical if the cache miss was caused by an eviction of active placements.
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

	// 3.0 Resolve user attrs against the rules that will actually evaluate.
	// Done after schedule+rule loading so we know exactly which CLEN fields
	// are needed → enables per-request delta fetch from CLEN.
	rulesForResolve := uniqueRulesFromFiltered(filtered)
	resolvedUserAttrs, resolveErr := s.resolveUserAttrs(ctx, customerInfo.TypeName(), customerInfo.Value(), rulesForResolve)
	if resolveErr != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "WARN",
			Message: fmt.Sprintf("Failed to resolve user attributes for customerId %q: %v", customerInfo.Value(), resolveErr),
		})
		return nil, resolveErr
	}

	// 3.0b When any rule resolving for this request is SALES_TARGET, fetch
	// lead offerings once for the whole request — N placements share the
	// same customer-level lead inventory. Failures are non-fatal: the
	// SALES_TARGET rule simply expands into zero entries.
	leads := s.fetchLeadsForRequest(ctx, customerInfo, channel, placementNames, rulesForResolve)

	// 3.1 Process each placement concurrently via errgroup.
	// Bounded concurrency limits parallelism during cache-miss storms
	// while still providing significant speedup.
	// Each goroutine writes to its own slot (no mutex needed), preserving
	// placementNames order in the merged result.
	const maxPlacementConcurrency = 10
	results := make([][]dto.ContentResult, len(placementNames))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxPlacementConcurrency)

	for i, name := range placementNames {
		g.Go(func() error {
			passing := s.evaluatePlacement(gctx, name, filtered[name], resolvedUserAttrs, leads)
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
// focused and testable independently. leads is the per-request CLEN lead
// inventory; SALES_TARGET rules expand into one entry per matching lead.
func (s *CMSDeliveryService) evaluatePlacement(
	ctx context.Context,
	placementName string,
	schedules []*entity.Schedule,
	resolvedUserAttrs map[string]json.RawMessage,
	leads []entity.Lead,
) []dto.ContentResult {
	var entries []dto.ContentResult
	// Evaluate the placement logic immediately if the evaluator and occurrenceRepo are configured; otherwise, return no results (cache miss).
	if s.evaluator != nil && s.occurrenceRepo != nil {
		passing, err := s.evaluator.Evaluate(ctx, placementName, schedules, resolvedUserAttrs, leads)
		if err != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "CMS-DELIVERY",
				Level:   "WARN",
				Message: fmt.Sprintf("Evaluate failed for %q: %v", placementName, err),
			})
			return nil
		}
		entries = passing
	} else {
		return nil
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("Evaluating %d logic entries for placement %q", len(entries), placementName),
	})
	return entries
}

// uniqueRulesFromFiltered collects deduplicated DecisionRule pointers from the
// per-placement schedule map produced after rule cache lookup. Used to scope
// resolveUserAttrs to only the rules that will actually evaluate this request.
func uniqueRulesFromFiltered(filtered map[string][]*entity.Schedule) []*entity.DecisionRule {
	if len(filtered) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{})
	out := make([]*entity.DecisionRule, 0)
	for _, scheds := range filtered {
		for _, sched := range scheds {
			if sched == nil || sched.DecisionRule == nil {
				continue
			}
			id := sched.DecisionRule.ID
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, sched.DecisionRule)
		}
	}
	return out
}

// fetchLeadsForRequest fetches CLEN lead offerings for the customer when at
// least one rule in this request is SALES_TARGET. Returns nil (no enrichment)
// when leadRepo is unset, the customer ID is not a CIS_ID, or no SALES_TARGET
// rule is in scope. Upstream errors are logged and swallowed so a CLEN outage
// can not fail the personalized-content path.
func (s *CMSDeliveryService) fetchLeadsForRequest(
	ctx context.Context,
	customerInfo *dto.CustomerRequest,
	channel string,
	placementNames []string,
	rules []*entity.DecisionRule,
) []entity.Lead {
	if s.leadRepo == nil || customerInfo == nil {
		return nil
	}
	if customerInfo.Type != dto.CustomerIdTypeCISID || customerInfo.Value() == "" {
		return nil
	}
	hasSalesTarget := false
	for _, r := range rules {
		if r != nil && r.Type == enums.DecisionTypeSalesTarget {
			hasSalesTarget = true
			break
		}
	}
	if !hasSalesTarget {
		return nil
	}
	leads, err := s.leadRepo.GetLeads(ctx, domainrepo.LeadQuery{
		CisID:      customerInfo.Value(),
		Channel:    channel,
		Placements: placementNames,
	})
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "WARN",
			Message: fmt.Sprintf("CLEN lead fetch failed for cis %s: %v", customerInfo.Value(), err),
		})
		return nil
	}
	return leads
}

// ---------------------------------------------------------------------------
// Background ticker — periodically evaluates all active placements in-process
// and populates L1 (CacheMemory) and L2 (version hashes).
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

	// Start background synchronization loops
	go s.runLoop(ctx)
	go s.subscribeToUpdates(ctx)

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("Background ticker and subscriber started (interval=%s)", s.tickInterval),
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
	if s.subCancel != nil {
		s.subCancel()
	}
	s.mu.Unlock()

	<-s.done
	if s.subDone != nil {
		<-s.subDone
	}
	logger.LSystem(context.Background(), entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: "Background ticker and subscriber stopped",
	})
	return nil
}

// runLoop fires evaluate immediately, then on every tick.
func (s *CMSDeliveryService) runLoop(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	// Initial pull
	s.evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evaluate(ctx)
		}
	}
}

// evaluate pulls active schedules from the backbone and synchronizes the local mirror.
func (s *CMSDeliveryService) evaluate(ctx context.Context, placementNames ...string) {
	var occurrences []*entity.ScheduleOccurrence
	var err error

	if len(placementNames) > 0 {
		occurrences, err = s.occurrenceRepo.ListActiveByPlacementsAt(ctx, placementNames, time.Now())
	} else {
		occurrences, err = s.occurrenceRepo.ListActiveAt(ctx, time.Now())
	}

	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "ERROR",
			Message: fmt.Sprintf("evaluate: failed to list active occurrences: %v", err),
		})
		return
	}

	// Targeted refresh cleanup: if a placement has no active schedules in DB,
	// we must clear its local cache entry.
	if len(placementNames) > 0 && len(occurrences) == 0 {
		for _, name := range placementNames {
			if s.cacheMemory != nil {
				s.cacheMemory.Schedules.Delete(cmsPlacementSchedulesKey(name))
				s.cacheMemory.VersionHashes.Delete(name)
				s.cacheMemory.LastSync.Set(name, time.Now(), s.resultTTL)
				// Clear metric for this placement
				s.mPlacementVersion.DeletePartialMatch(prometheus.Labels{"placement_name": name})
			}
		}
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
		// If no groups found, clear all schedules and rules in cache
		if s.cacheMemory != nil {
			s.cacheMemory.UpdateSchedules(nil, nil, nil, s.resultTTL, s.updateVersionMetric)
		}
		return
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("evaluate: processing %d placements (%d schedules)", len(groups), len(schedules)),
	})

	// Fetch all rules for these schedules to ensure rule cache is also mirrored.
	scheduleIDs := make([]uuid.UUID, 0, len(schedules))
	for _, sched := range schedules {
		scheduleIDs = append(scheduleIDs, sched.ID)
	}
	rules, err := s.decisionRepo.GetDecisionRuleByScheduleIDs(ctx, scheduleIDs)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "WARN",
			Message: fmt.Sprintf("evaluate: failed to fetch rules for %d schedules: %v", len(scheduleIDs), err),
		})
		// Continue with schedules only if rules can't be fetched bulk;
		// they will be fetched on demand.
	}

	// Prepare new cache sets
	newSchedules := make(map[string][]*entity.Schedule)
	newRules := make(map[string]*entity.DecisionRule)
	newVersions := make(map[string]string)

	for placementName, g := range groups {
		key := cmsPlacementSchedulesKey(placementName)
		newSchedules[key] = g.schedules

		newVersions[placementName] = computePlacementHash(g.schedules)
	}
	if len(rules) > 0 {
		for _, rule := range rules {
			newRules[ruleDecisionCacheKey(rule.ID.String())] = rule
		}
	}

	// Perform re-population
	if s.cacheMemory != nil {
		if len(placementNames) > 0 {
			// In targeted refresh, we only update specific keys.
			// We don't use UpdateSchedules because that method performs a full-slice delete of other keys.
			// Instead, we update these specific ones and their LastSync.
			now := time.Now()
			for name, g := range groups {
				key := cmsPlacementSchedulesKey(name)
				s.cacheMemory.Schedules.Set(key, g.schedules, s.resultTTL)
				s.cacheMemory.LastSync.Set(name, now, s.resultTTL)

				hashStr := computePlacementHash(g.schedules)
				s.cacheMemory.VersionHashes.Set(name, hashStr, s.resultTTL)

				// Update metrics
				s.updateVersionMetric(name, hashStr)
			}
			// Update rules for these specific schedules
			for _, rule := range rules {
				s.cacheMemory.DecisionRule.Set(ruleDecisionCacheKey(rule.ID.String()), rule, s.resultTTL)
			}
			// Prune rules not referenced by current in-memory schedules
			s.cacheMemory.PruneOrphanedRules()
		} else {
			// Full-slice re-population
			s.cacheMemory.UpdateSchedules(newSchedules, newRules, newVersions, s.resultTTL, s.updateVersionMetric)
		}
	}
}

// updateVersionMetric updates the Prometheus gauge for a placement's version.
func (s *CMSDeliveryService) updateVersionMetric(placement, newVersion string) {
	if s.mPlacementVersion == nil {
		return
	}

	// Retrieve old version to clean up labels
	if s.cacheMemory != nil && s.cacheMemory.VersionHashes != nil {
		if oldVersion, ok := s.cacheMemory.VersionHashes.Get(placement); ok && oldVersion != newVersion {
			s.mPlacementVersion.DeleteLabelValues(placement, oldVersion)
		}
	}

	s.mPlacementVersion.WithLabelValues(placement, newVersion).Set(1)
}

// subscribeToUpdates listens on the Redis Pub/Sub channel for sync pings.
func (s *CMSDeliveryService) subscribeToUpdates(ctx context.Context) {
	s.mu.Lock()
	s.subCtx, s.subCancel = context.WithCancel(ctx)
	s.subDone = make(chan struct{})
	s.mu.Unlock()

	defer close(s.subDone)

	const channel = pubsub.ChannelCMSSyncPing
	msgs, err := s.cacheRepo.Subscribe(s.subCtx, channel)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "ERROR",
			Message: fmt.Sprintf("subscriber: failed to subscribe to %q: %v", channel, err),
		})
		return
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("subscriber: listening on %q", channel),
	})

	for {
		select {
		case <-s.subCtx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			// Try to unmarshal structured message
			var ping SyncPingMessage
			if err := json.Unmarshal([]byte(msg), &ping); err != nil {
				logger.LSystem(s.subCtx, entity.SystemLog{
					Service: "CMS-DELIVERY",
					Level:   "INFO",
					Message: fmt.Sprintf("subscriber: received raw ping %q, triggering full evaluate", msg),
				})
			} else {
				// Priority 2: Consistency Check
				// If we have a version hash, check if we already have this version.
				if s.cacheMemory != nil && ping.VersionHash != "" {
					if localHash, ok := s.cacheMemory.VersionHashes.Get(ping.PlacementName); ok && localHash == ping.VersionHash {
						logger.LSystem(s.subCtx, entity.SystemLog{
							Service: "CMS-DELIVERY",
							Level:   "INFO",
							Message: fmt.Sprintf("subscriber: version %s for %q already mirrored, skipping pull", ping.VersionHash, ping.PlacementName),
						})
						continue
					}
				}
				logger.LSystem(s.subCtx, entity.SystemLog{
					Service: "CMS-DELIVERY",
					Level:   "INFO",
					Message: fmt.Sprintf("subscriber: received ping for %q (version %s), triggering evaluate", ping.PlacementName, ping.VersionHash),
				})
			}

			// Priority 2: Jittered Synchronization
			// Flatten the load curve during cluster-wide updates.
			// Start with a minimum 50ms buffer to avoid 0ms collisions.
			jitter := time.Duration(rand.Intn(450)+50) * time.Millisecond
			select {
			case <-s.subCtx.Done():
				return
			case <-time.After(jitter):
				// Re-verify version after jitter to prevent race condition between multiple pods
				if s.cacheMemory != nil && ping.VersionHash != "" {
					if localHash, ok := s.cacheMemory.VersionHashes.Get(ping.PlacementName); ok && localHash == ping.VersionHash {
						continue
					}
				}

				// Targeted invalidation: when the publisher names a specific
				// decision rule (e.g. on activate), drop both cache entries
				// before re-evaluating so the next read can never observe a
				// stale `rule:{id}` mirror, even briefly.
				if s.cacheMemory != nil && ping.DecisionRuleID != "" {
					s.cacheMemory.DecisionRule.Delete(ruleDecisionCacheKey(ping.DecisionRuleID))
					if ping.PlacementName != "" {
						s.cacheMemory.Schedules.Delete(cmsPlacementSchedulesKey(ping.PlacementName))
					}
				}

				if ping.PlacementName != "" {
					s.evaluate(s.subCtx, ping.PlacementName)
				} else {
					s.evaluate(s.subCtx)
				}
			}
		}
	}
}

// setCache writes value to both the in-memory cache (if configured) and Redis.
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

// getCache retrieves a value by key, checking in-memory cache first then falling back to Redis.
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
