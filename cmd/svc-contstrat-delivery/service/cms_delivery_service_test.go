package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/cache"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// newTestMemCache builds a *MemoryCache for tests and registers cleanup.
func newTestMemCache(t *testing.T, ns string) *MemoryCache {
	t.Helper()
	s := cache.NewCacheMemory[[]*entity.Schedule](ns+"_schedules", 0.95, time.Hour)
	r := cache.NewCacheMemory[*entity.DecisionRule](ns+"_rules", 0.95, time.Hour)
	t.Cleanup(s.Stop)
	t.Cleanup(r.Stop)
	return &MemoryCache{Schedules: s, DecisionRule: r}
}

// newSvcCacheOnly builds a CMSDeliveryService with only a cache repo (no evaluator).
func newSvcCacheOnly(cacheRepo *mockCacheRepo, mem *MemoryCache) *CMSDeliveryService {
	return NewCMSDeliveryService(cacheRepo, &mockOccurrenceRepo{}, &mockDecisionRuleRepo{}, nil, mem, time.Hour, 0)
}

// newSvcWithFallback builds a CMSDeliveryService with cache, occurrence repo, and evaluator.
func newSvcWithFallback(
	cacheRepo *mockCacheRepo,
	occurrenceRepo *mockOccurrenceRepo,
	eval RuntimeEvaluator,
	mem *MemoryCache,
) *CMSDeliveryService {
	return NewCMSDeliveryService(cacheRepo, occurrenceRepo, &mockDecisionRuleRepo{}, eval, mem, time.Hour, 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// Mock types (package-local)
// ─────────────────────────────────────────────────────────────────────────────

// mockCacheRepo is a minimal mock for domainrepo.CacheRepository.
type mockCacheRepo struct {
	getFn    func(ctx context.Context, key string) (string, error)
	setFn    func(ctx context.Context, key string, val string, exp time.Duration) error
	deleteFn func(ctx context.Context, key string) error
	flushFn  func(ctx context.Context) error
}

func (m *mockCacheRepo) Get(ctx context.Context, key string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return "", errors.New("not found")
}
func (m *mockCacheRepo) Set(ctx context.Context, key string, val string, exp time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, val, exp)
	}
	return nil
}
func (m *mockCacheRepo) HGet(_ context.Context, _, _ string) (string, error) { return "", nil }
func (m *mockCacheRepo) HSet(_ context.Context, _, _, _ string) error        { return nil }
func (m *mockCacheRepo) FlushDB(ctx context.Context) error {
	if m.flushFn != nil {
		return m.flushFn(ctx)
	}
	return nil
}
func (m *mockCacheRepo) GetSet(ctx context.Context, key string, exp time.Duration, loader func(context.Context) (string, error)) (string, error) {
	if m.getFn != nil {
		if value, err := m.getFn(ctx, key); err == nil {
			return value, nil
		}
	}
	if loader == nil {
		return "", errors.New("not found")
	}
	value, err := loader(ctx)
	if err != nil {
		return "", err
	}
	if m.setFn != nil {
		if err := m.setFn(ctx, key, value, exp); err != nil {
			return "", err
		}
	}
	return value, nil
}
func (m *mockCacheRepo) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

// mockOccurrenceRepo is a minimal mock for domainrepo.ScheduleOccurrenceRepository.
type mockOccurrenceRepo struct {
	listActiveAtFn func(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error)
}

func (m *mockOccurrenceRepo) ListActiveAt(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error) {
	if m.listActiveAtFn != nil {
		return m.listActiveAtFn(ctx, at)
	}
	return nil, nil
}
func (m *mockOccurrenceRepo) UpsertOccurrences(_ context.Context, _ []*entity.ScheduleOccurrence) error {
	return nil
}
func (m *mockOccurrenceRepo) DeleteFutureByScheduleID(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockOccurrenceRepo) DeletePastOccurrences(_ context.Context, _ time.Time) error { return nil }
func (m *mockOccurrenceRepo) ListByScheduleID(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.ScheduleOccurrence, int64, error) {
	return nil, 0, nil
}

// mockDecisionRuleRepo is a minimal mock for domainrepo.DecisionRuleRepository.
type mockDecisionRuleRepo struct {
	getFn  func(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error)
	getsFn func(ctx context.Context, scheduleIDs []uuid.UUID) (map[uuid.UUID]*entity.DecisionRule, error)
}

func (m *mockDecisionRuleRepo) GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error) {
	if m.getFn != nil {
		return m.getFn(ctx, scheduleID)
	}
	return nil, nil
}

func (m *mockDecisionRuleRepo) GetDecisionRuleByScheduleIDs(ctx context.Context, scheduleIDs []uuid.UUID) (map[uuid.UUID]*entity.DecisionRule, error) {
	if m.getsFn != nil {
		return m.getsFn(ctx, scheduleIDs)
	}
	return nil, nil
}

// mockRuntimeEvaluator is a minimal mock for RuntimeEvaluator.
type mockRuntimeEvaluator struct {
	evaluateFn func(ctx context.Context, name string, schedules []*entity.Schedule, userAttrs map[string]json.RawMessage) ([]dto.ContentResult, error)
}

func (m *mockRuntimeEvaluator) Evaluate(
	ctx context.Context,
	name string,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) ([]dto.ContentResult, error) {
	if m.evaluateFn != nil {
		return m.evaluateFn(ctx, name, schedules, userAttrs)
	}
	return nil, nil
}

// compile-time guard.
var _ RuntimeEvaluator = (*mockRuntimeEvaluator)(nil)

// ─────────────────────────────────────────────────────────────────────────────
// FlushCache tests
// ─────────────────────────────────────────────────────────────────────────────

// TestCMSDeliveryService_FlushCache_Selective verifies that FlushCache clears
// in-memory schedule entries for each named placement without touching Redis.
func TestCMSDeliveryService_FlushCache_Selective(t *testing.T) {
	t.Parallel()

	deletedKeys := []string{}
	flushDBCalled := false
	mem := newTestMemCache(t, "flush_cache_selective_test")
	mem.Schedules.Set(cmsPlacementSchedulesKey("hero"), []*entity.Schedule{}, time.Hour)
	mem.Schedules.Set(cmsPlacementSchedulesKey("sidebar"), []*entity.Schedule{}, time.Hour)

	svc := NewCMSDeliveryService(&mockCacheRepo{
		deleteFn: func(_ context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
		flushFn: func(_ context.Context) error {
			flushDBCalled = true
			return nil
		},
	}, nil, nil, nil, mem, time.Hour, 0)

	err := svc.FlushCache(context.Background(), []string{"hero", "sidebar"}, false)

	require.NoError(t, err)
	assert.False(t, flushDBCalled)
	assert.Empty(t, deletedKeys, "selective flush should not call Redis Delete")
	_, ok := mem.Schedules.Get(cmsPlacementSchedulesKey("hero"))
	assert.False(t, ok, "hero schedules should be evicted from in-memory cache")
	_, ok = mem.Schedules.Get(cmsPlacementSchedulesKey("sidebar"))
	assert.False(t, ok, "sidebar schedules should be evicted from in-memory cache")
}

// TestCMSDeliveryService_FlushCache_All verifies that FlushCache with nil/empty
// names calls FlushDB instead of Delete.
func TestCMSDeliveryService_FlushCache_All(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	flushDBCalled := false
	mem := newTestMemCache(t, "flush_cache_all_test")

	svc := NewCMSDeliveryService(&mockCacheRepo{
		deleteFn: func(_ context.Context, _ string) error {
			deleteCalled = true
			return nil
		},
		flushFn: func(_ context.Context) error {
			flushDBCalled = true
			return nil
		},
	}, nil, nil, nil, mem, time.Hour, 0)

	// nil placements → FlushDB
	require.NoError(t, svc.FlushCache(context.Background(), nil, false))
	assert.True(t, flushDBCalled)
	assert.False(t, deleteCalled)

	// empty slice → FlushDB
	flushDBCalled = false
	require.NoError(t, svc.FlushCache(context.Background(), []string{}, false))
	assert.True(t, flushDBCalled)
	assert.False(t, deleteCalled)
}

// ─────────────────────────────────────────────────────────────────────────────
// GetPersonalizedContent tests
// ─────────────────────────────────────────────────────────────────────────────

// TestGetPersonalizedContent_EvaluatorFallback verifies that a personalized cache
// miss triggers RuntimeEvaluator.Evaluate with the schedules from in-memory cache
// and returns the evaluator's results.
func TestGetPersonalizedContent_EvaluatorFallback(t *testing.T) {
	t.Parallel()

	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/content/hero",
		RuleSetType:    "MASS",
		LogicEval:      true,
		Score:          0.8,
	}

	mem := newTestMemCache(t, "evaluator_fallback")
	rule := &entity.DecisionRule{BaseModel: entity.BaseModel{ID: uuid.New()}}
	sched := &entity.Schedule{
		Placement:    &entity.Placement{PlacementName: "hero"},
		DecisionRule: rule,
	}
	mem.Schedules.Set(cmsPlacementSchedulesKey("hero"), []*entity.Schedule{sched}, time.Hour)

	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("miss")
			},
		},
		&mockOccurrenceRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, name string, schedules []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				assert.Equal(t, "hero", name)
				require.Len(t, schedules, 1)
				return []dto.ContentResult{entry}, nil
			},
		},
		mem,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis1", "user1", []string{"hero"}, map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/content/hero", result[0].ContentPath)
}

// TestGetPersonalizedContent_EvaluatorFails verifies graceful degradation:
// evaluator failure returns an empty result without propagating the error.
func TestGetPersonalizedContent_EvaluatorFails(t *testing.T) {
	t.Parallel()

	mem := newTestMemCache(t, "evaluator_fails")

	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("miss")
			},
		},
		&mockOccurrenceRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return nil, errors.New("rpc error: unavailable")
			},
		},
		mem,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis1", "user1", []string{"hero"}, map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetPersonalizedContent_NoSchedulesInMemory verifies that a placement with
// no schedules in the in-memory cache returns an empty result when no evaluator is set.
func TestGetPersonalizedContent_NoSchedulesInMemory(t *testing.T) {
	t.Parallel()

	mem := newTestMemCache(t, "no_schedules")

	svc := newSvcCacheOnly(&mockCacheRepo{
		getFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("miss")
		},
	}, mem)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis1", "user1", []string{"unknown-placement"}, nil,
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetPersonalizedContent_LoadsUserAttrsFromRedisByCISID verifies that the
// delivery service resolves a cis_id:{cisID} JSON payload from Redis and passes
// the merged attributes to the evaluator.
func TestGetPersonalizedContent_LoadsUserAttrsFromRedisByCISID(t *testing.T) {
	t.Parallel()

	condAttrID := uuid.New()
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/wealth/vip",
		LogicEval:      true,
		Score:          9.5,
	}

	userAttrsData, _ := json.Marshal(map[string]any{
		condAttrID.String(): "balanced",
	})

	stored := map[string]string{
		cmsUserAttrsKey("cis42"): string(userAttrsData),
	}

	mem := newTestMemCache(t, "user_attrs_from_redis")

	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
		},
		&mockOccurrenceRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, userAttrs map[string]json.RawMessage) ([]dto.ContentResult, error) {
				_, ok := userAttrs[condAttrID.String()]
				assert.True(t, ok, "expected resolved user attr to be passed to evaluator")
				return []dto.ContentResult{entry}, nil
			},
		},
		mem,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis42", "user88", []string{"wealth-banner"}, nil,
	)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/wealth/vip", result[0].ContentPath)
}

// TestGetPersonalizedContent_CacheMissTriggersEvaluate verifies that when the
// schedule cache is empty (e.g. TTL expired), GetPersonalizedContent calls
// evaluate() to re-populate the cache and returns non-empty results.
func TestGetPersonalizedContent_CacheMissTriggersEvaluate(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	schedID := uuid.New()
	rule := &entity.DecisionRule{BaseModel: entity.BaseModel{ID: ruleID}}
	placement := &entity.Placement{PlacementName: "hero"}
	sched := &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: schedID},
		Placement:      placement,
		DecisionRule:   rule,
		DecisionRuleID: ruleID,
	}
	occ := &entity.ScheduleOccurrence{
		ScheduleID: schedID,
		Schedule:   sched,
	}

	entry := dto.ContentResult{
		ContentPath: "/hero/banner",
		LogicEval:   true,
		Score:       1.0,
	}

	mem := newTestMemCache(t, "cache_miss_triggers_evaluate")
	// Cache starts empty — simulates TTL expiry.

	listCalled := false
	svc := NewCMSDeliveryService(
		&mockCacheRepo{},
		&mockOccurrenceRepo{
			listActiveAtFn: func(_ context.Context, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
				listCalled = true
				return []*entity.ScheduleOccurrence{occ}, nil
			},
		},
		&mockDecisionRuleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return []dto.ContentResult{entry}, nil
			},
		},
		mem, time.Hour, 0,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis1", "user1", []string{"hero"}, map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	assert.True(t, listCalled, "expected ListActiveAt to be called as evaluate fallback")
	require.Len(t, result, 1)
	assert.Equal(t, "/hero/banner", result[0].ContentPath)
}

// TestGetPersonalizedContent_CacheMissPersistsGracefully verifies that when
// evaluate() finds no active occurrences, the placement is skipped and an
// empty result is returned without error.
func TestGetPersonalizedContent_CacheMissPersistsGracefully(t *testing.T) {
	t.Parallel()

	mem := newTestMemCache(t, "cache_miss_persists")

	svc := NewCMSDeliveryService(
		&mockCacheRepo{},
		&mockOccurrenceRepo{
			listActiveAtFn: func(_ context.Context, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
				return nil, nil // no active occurrences
			},
		},
		&mockDecisionRuleRepo{},
		&mockRuntimeEvaluator{},
		mem, time.Hour, 0,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis1", "user1", []string{"hero"}, map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}
