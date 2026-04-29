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
	v := cache.NewCacheMemory[string](ns+"_versions", 0.95, time.Hour)
	l := cache.NewCacheMemory[time.Time](ns+"_syncs", 0.95, time.Hour)
	t.Cleanup(s.Stop)
	t.Cleanup(r.Stop)
	t.Cleanup(v.Stop)
	t.Cleanup(l.Stop)
	return &MemoryCache{Schedules: s, DecisionRule: r, VersionHashes: v, LastSync: l}
}

// newSvcCacheOnly builds a CMSDeliveryService with only a cache repo (no evaluator).
func newSvcCacheOnly(cacheRepo *mockCacheRepo, mem *MemoryCache) *CMSDeliveryService {
	return NewCMSDeliveryService(cacheRepo, &mockOccurrenceRepo{}, &mockDecisionRuleRepo{}, nil, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)
}

// newSvcWithFallback builds a CMSDeliveryService with cache, occurrence repo, and evaluator.
func newSvcWithFallback(
	cacheRepo *mockCacheRepo,
	occurrenceRepo *mockOccurrenceRepo,
	eval RuntimeEvaluator,
	mem *MemoryCache,
) *CMSDeliveryService {
	return NewCMSDeliveryService(cacheRepo, occurrenceRepo, &mockDecisionRuleRepo{}, eval, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)
}

// (Pre-merge note: the old gRPC-fallback / cmsPlacementKey-based tests that
// lived here were dropped during the dev merge — those APIs were replaced by
// the in-process LocalEvaluator and the typed MemoryCache struct. Equivalent
// coverage is provided below by TestGetPersonalizedContent_EvaluatorFallback /
// _CacheMissTriggersEvaluate and TestCMSDeliveryService_FlushCache_Selective /
// _All.)

// ─────────────────────────────────────────────────────────────────────────────
// Mock types (package-local)
// ─────────────────────────────────────────────────────────────────────────────

// mockCacheRepo is a minimal mock for domainrepo.CacheRepository.
type mockCacheRepo struct {
	getFn       func(ctx context.Context, key string) (string, error)
	setFn       func(ctx context.Context, key string, val string, exp time.Duration) error
	deleteFn    func(ctx context.Context, key string) error
	flushFn     func(ctx context.Context) error
	subscribeFn func(ctx context.Context, channel string) (<-chan string, error)
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
func (m *mockCacheRepo) Subscribe(ctx context.Context, channel string) (<-chan string, error) {
	if m.subscribeFn != nil {
		return m.subscribeFn(ctx, channel)
	}
	return make(chan string), nil
}
func (m *mockCacheRepo) Publish(_ context.Context, _ string, _ string) error { return nil }

// mockOccurrenceRepo is a minimal mock for domainrepo.ScheduleOccurrenceRepository.
type mockOccurrenceRepo struct {
	listActiveAtFn             func(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error)
	listActiveByPlacementsAtFn func(ctx context.Context, placementNames []string, at time.Time) ([]*entity.ScheduleOccurrence, error)
}

func (m *mockOccurrenceRepo) ListActiveAt(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error) {
	if m.listActiveAtFn != nil {
		return m.listActiveAtFn(ctx, at)
	}
	return nil, nil
}
func (m *mockOccurrenceRepo) ListActiveByPlacementsAt(ctx context.Context, placementNames []string, at time.Time) ([]*entity.ScheduleOccurrence, error) {
	if m.listActiveByPlacementsAtFn != nil {
		return m.listActiveByPlacementsAtFn(ctx, placementNames, at)
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
func (m *mockOccurrenceRepo) ExpireEndedOccurrences(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
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
	evaluateFn func(ctx context.Context, name string, schedules []*entity.Schedule, userAttrs map[string]json.RawMessage, leads []entity.Lead) ([]dto.ContentResult, error)
}

func (m *mockRuntimeEvaluator) Evaluate(
	ctx context.Context,
	name string,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
	leads []entity.Lead,
) ([]dto.ContentResult, error) {
	if m.evaluateFn != nil {
		return m.evaluateFn(ctx, name, schedules, userAttrs, leads)
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
	}, nil, nil, nil, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)

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
	}, nil, nil, nil, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)

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
			evaluateFn: func(_ context.Context, name string, schedules []*entity.Schedule, _ map[string]json.RawMessage, _ []entity.Lead) ([]dto.ContentResult, error) {
				assert.Equal(t, "hero", name)
				require.Len(t, schedules, 1)
				return []dto.ContentResult{entry}, nil
			},
		},
		mem,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis1"}, "", []string{"hero"},
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
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage, _ []entity.Lead) ([]dto.ContentResult, error) {
				return nil, errors.New("rpc error: unavailable")
			},
		},
		mem,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis1"}, "", []string{"hero"},
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
		context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis1"}, "", []string{"unknown-placement"},
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetPersonalizedContent_LoadsUserAttrsFromRedisByCISID verifies that the
// delivery service resolves a customer_profile:{customerType}:{customerID} per-datasource JSON payload from
// Redis, transforms each (datasource, fieldName) into its Attribute UUID via
// the attribute repository, and passes the flat UUID-keyed map to the evaluator.
func TestGetPersonalizedContent_LoadsUserAttrsFromRedisByCISID(t *testing.T) {
	t.Parallel()

	condAttrID := uuid.New()
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/wealth/vip",
		LogicEval:      true,
		Score:          9.5,
	}

	// Cache stores the per-datasource shape; attribute repo maps the field
	// name to the UUID the evaluator expects.
	stored := map[string]string{
		cmsUserAttrsKey("CIS_ID", "cis42"): `{"cst_info_prfl_dly":{"investor_type":"balanced"}}`,
	}
	attrRepo := &fakeAttributeRepo{byDatasource: map[string][]*entity.Attribute{
		"cst_info_prfl_dly": {
			{BaseModel: entity.BaseModel{ID: condAttrID}, FieldName: "investor_type"},
		},
	}}

	mem := newTestMemCache(t, "user_attrs_from_redis")

	svc := NewCMSDeliveryService(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
		},
		&mockOccurrenceRepo{}, &mockDecisionRuleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, userAttrs map[string]json.RawMessage, _ []entity.Lead) ([]dto.ContentResult, error) {
				_, ok := userAttrs[condAttrID.String()]
				assert.True(t, ok, "expected resolved user attr to be passed to evaluator")
				return []dto.ContentResult{entry}, nil
			},
		},
		mem,
		time.Hour, 0,
		nil,
		nil, CustomerProfileEnrichConfig{},
		nil, attrRepo,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis42"}, "", []string{"wealth-banner"},
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
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage, _ []entity.Lead) ([]dto.ContentResult, error) {
				return []dto.ContentResult{entry}, nil
			},
		},
		mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis1"}, "", []string{"hero"},
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
		mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis1"}, "", []string{"hero"},
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// GetCacheKeys tests
// ─────────────────────────────────────────────────────────────────────────────

// TestGetCacheKeys_NilCache verifies GetCacheKeys returns an empty slice when no in-memory cache is configured.
func TestGetCacheKeys_NilCache(t *testing.T) {
	t.Parallel()
	svc := NewCMSDeliveryService(&mockCacheRepo{}, nil, nil, nil, nil, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)

	keys, err := svc.GetCacheKeys(context.Background())
	require.NoError(t, err)
	assert.Empty(t, keys)
}

// TestGetCacheKeys_WithEntries verifies GetCacheKeys returns keys from both Schedules and DecisionRule caches.
func TestGetCacheKeys_WithEntries(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "get_cache_keys_test")
	schedKey := cmsPlacementSchedulesKey("hero")
	mem.Schedules.Set(schedKey, []*entity.Schedule{}, time.Hour)
	ruleID := uuid.New()
	ruleKey := ruleDecisionCacheKey(ruleID.String())
	mem.DecisionRule.Set(ruleKey, &entity.DecisionRule{BaseModel: entity.BaseModel{ID: ruleID}}, time.Hour)

	svc := newSvcCacheOnly(&mockCacheRepo{}, mem)

	keys, err := svc.GetCacheKeys(context.Background())
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, schedKey)
	assert.Contains(t, keys, ruleKey)
}

// ─────────────────────────────────────────────────────────────────────────────
// GetCacheValue tests
// ─────────────────────────────────────────────────────────────────────────────

// TestGetCacheValue_ScheduleKey verifies that GetCacheValue returns JSON-encoded schedules for a schedules: key.
func TestGetCacheValue_ScheduleKey(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "get_cache_value_schedule_test")
	key := cmsPlacementSchedulesKey("hero")
	mem.Schedules.Set(key, []*entity.Schedule{}, time.Hour)

	svc := newSvcCacheOnly(&mockCacheRepo{}, mem)

	val, err := svc.GetCacheValue(context.Background(), key)
	require.NoError(t, err)
	assert.NotNil(t, val)
	// Confirm the value is valid JSON.
	var decoded []*entity.Schedule
	require.NoError(t, json.Unmarshal(val, &decoded))
}

// TestGetCacheValue_RuleKey verifies that GetCacheValue returns JSON-encoded rule for a rule: key.
func TestGetCacheValue_RuleKey(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "get_cache_value_rule_test")
	ruleID := uuid.New()
	key := ruleDecisionCacheKey(ruleID.String())
	mem.DecisionRule.Set(key, &entity.DecisionRule{BaseModel: entity.BaseModel{ID: ruleID}}, time.Hour)

	svc := newSvcCacheOnly(&mockCacheRepo{}, mem)

	val, err := svc.GetCacheValue(context.Background(), key)
	require.NoError(t, err)
	assert.NotNil(t, val)
	// Confirm the value is valid JSON.
	var decoded entity.DecisionRule
	require.NoError(t, json.Unmarshal(val, &decoded))
}

// TestGetCacheValue_UnsupportedPrefix verifies that GetCacheValue returns an error for an unrecognised key prefix.
func TestGetCacheValue_UnsupportedPrefix(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "get_cache_value_unsupported_test")
	svc := newSvcCacheOnly(&mockCacheRepo{}, mem)

	_, err := svc.GetCacheValue(context.Background(), "cms:placement:hero")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key prefix")
}

// TestGetCacheValue_KeyNotFound verifies that GetCacheValue returns an error when the key is absent from the cache.
func TestGetCacheValue_KeyNotFound(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "get_cache_value_not_found_test")
	svc := newSvcCacheOnly(&mockCacheRepo{}, mem)

	_, err := svc.GetCacheValue(context.Background(), ruleDecisionCacheKey("nonexistent"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ─────────────────────────────────────────────────────────────────────────────
// GetCacheStatus tests
// ─────────────────────────────────────────────────────────────────────────────

// TestGetCacheStatus_NilCache verifies GetCacheStatus returns false/0 when no in-memory cache is configured.
func TestGetCacheStatus_NilCache(t *testing.T) {
	t.Parallel()
	svc := NewCMSDeliveryService(&mockCacheRepo{}, nil, nil, nil, nil, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)

	pressure, pct, err := svc.GetCacheStatus(context.Background())
	require.NoError(t, err)
	assert.False(t, pressure)
	assert.Equal(t, 0.0, pct)
}

// TestGetCacheStatus_WithCache verifies GetCacheStatus returns values from the in-memory cache without error.
func TestGetCacheStatus_WithCache(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "get_cache_status_test")
	svc := newSvcCacheOnly(&mockCacheRepo{}, mem)

	_, _, err := svc.GetCacheStatus(context.Background())
	require.NoError(t, err)
}

// TestMemoryCache_UpdateSchedules verifies that UpdateSchedules correctly synchronizes
// the local cache with the provided sets, including deleting stale keys.
func TestMemoryCache_UpdateSchedules(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "update_schedules_test")

	// Setup initial state
	oldSchedKey := cmsPlacementSchedulesKey("old-placement")
	keepSchedKey := cmsPlacementSchedulesKey("keep-placement")
	oldRuleKey := ruleDecisionCacheKey(uuid.New().String())
	keepRuleKey := ruleDecisionCacheKey(uuid.New().String())

	mem.Schedules.Set(oldSchedKey, []*entity.Schedule{}, time.Hour)
	mem.Schedules.Set(keepSchedKey, []*entity.Schedule{}, time.Hour)
	mem.DecisionRule.Set(oldRuleKey, &entity.DecisionRule{}, time.Hour)
	mem.DecisionRule.Set(keepRuleKey, &entity.DecisionRule{}, time.Hour)

	// New state
	newSchedKey := cmsPlacementSchedulesKey("new-placement")
	newRuleKey := ruleDecisionCacheKey(uuid.New().String())

	newSchedules := map[string][]*entity.Schedule{
		keepSchedKey: {},
		newSchedKey:  {},
	}
	newRules := map[string]*entity.DecisionRule{
		keepRuleKey: {},
		newRuleKey:  {},
	}
	newVersions := map[string]string{
		"keep-placement": "v2",
		"new-placement":  "v1",
	}

	mem.UpdateSchedules(newSchedules, newRules, newVersions, time.Hour, nil)

	// Verify Schedules
	_, ok := mem.Schedules.Get(oldSchedKey)
	assert.False(t, ok, "old schedule key should be deleted")
	_, ok = mem.Schedules.Get(keepSchedKey)
	assert.True(t, ok, "keep schedule key should be present")
	_, ok = mem.Schedules.Get(newSchedKey)
	assert.True(t, ok, "new schedule key should be present")

	// Verify Rules
	_, ok = mem.DecisionRule.Get(oldRuleKey)
	assert.False(t, ok, "old rule key should be deleted")
	_, ok = mem.DecisionRule.Get(keepRuleKey)
	assert.True(t, ok, "keep rule key should be present")
	_, ok = mem.DecisionRule.Get(newRuleKey)
	assert.True(t, ok, "new rule key should be present")

	// Verify Versions
	v, ok := mem.VersionHashes.Get("keep-placement")
	assert.True(t, ok)
	assert.Equal(t, "v2", v)

	// Verify LastSync
	ls, ok := mem.LastSync.Get("keep-placement")
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now(), ls, 5*time.Second)
}

// TestGetPersonalizedContent_StalenessFailFast verifies that GetPersonalizedContent returns
// an integrity error when the requested placement's mirror is stale.
func TestGetPersonalizedContent_StalenessFailFast(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "staleness_fail_fast")
	placementName := "hero"

	// Mock stale state: last sync was 1 hour ago
	mem.LastSync.Set(placementName, time.Now().Add(-1*time.Hour), time.Hour)
	mem.Schedules.Set(cmsPlacementSchedulesKey(placementName), []*entity.Schedule{}, time.Hour)

	// Service with 1m tick interval -> 2m threshold
	svc := NewCMSDeliveryService(&mockCacheRepo{}, &mockOccurrenceRepo{
		listActiveAtFn: func(_ context.Context, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
			return nil, nil // Return nothing to simulate failed refresh/sync
		},
	}, nil, nil, mem, time.Hour, time.Minute, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)

	_, err := svc.GetPersonalizedContent(context.Background(), &dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "cis1"}, "", []string{placementName})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "data integrity error")
	assert.Contains(t, err.Error(), placementName)
}

// TestCMSDeliveryService_TargetedEvaluate verifies that evaluate only fetches specific placements
// when requested.
func TestCMSDeliveryService_TargetedEvaluate(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "targeted_evaluate")
	placementName := "hero"

	calledTargeted := false
	repo := &mockOccurrenceRepo{
		listActiveByPlacementsAtFn: func(ctx context.Context, placementNames []string, at time.Time) ([]*entity.ScheduleOccurrence, error) {
			calledTargeted = true
			assert.Equal(t, []string{placementName}, placementNames)
			return []*entity.ScheduleOccurrence{
				{
					Schedule: &entity.Schedule{
						BaseModel: entity.BaseModel{ID: uuid.New(), UpdatedAt: time.Now()},
						Placement: &entity.Placement{PlacementName: placementName},
					},
				},
			}, nil
		},
	}

	svc := NewCMSDeliveryService(&mockCacheRepo{}, repo, &mockDecisionRuleRepo{}, nil, mem, time.Hour, time.Minute, nil, nil, CustomerProfileEnrichConfig{}, nil, nil)

	svc.evaluate(context.Background(), placementName)

	assert.True(t, calledTargeted, "targeted fetch should have been called")
	_, ok := mem.Schedules.Get(cmsPlacementSchedulesKey(placementName))
	assert.True(t, ok, "cache should be populated for targeted placement")
	_, ok = mem.LastSync.Get(placementName)
	assert.True(t, ok, "LastSync should be updated for targeted placement")
}

// TestMemoryCache_PruneOrphanedRules verifies that rules are evicted when no longer referenced.
func TestMemoryCache_PruneOrphanedRules(t *testing.T) {
	t.Parallel()
	mem := newTestMemCache(t, "prune_orphaned_rules")

	ruleID1 := uuid.New()
	ruleID2 := uuid.New()
	placement1 := "hero"
	placement2 := "banner"

	// Initial state: 2 placements, 2 rules
	schedules1 := []*entity.Schedule{
		{BaseModel: entity.BaseModel{ID: uuid.New()}, DecisionRuleID: ruleID1},
	}
	schedules2 := []*entity.Schedule{
		{BaseModel: entity.BaseModel{ID: uuid.New()}, DecisionRuleID: ruleID2},
	}

	mem.Schedules.Set(cmsPlacementSchedulesKey(placement1), schedules1, time.Hour)
	mem.Schedules.Set(cmsPlacementSchedulesKey(placement2), schedules2, time.Hour)
	mem.DecisionRule.Set(ruleDecisionCacheKey(ruleID1.String()), &entity.DecisionRule{}, time.Hour)
	mem.DecisionRule.Set(ruleDecisionCacheKey(ruleID2.String()), &entity.DecisionRule{}, time.Hour)

	// Verify both rules exist
	_, ok := mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID1.String()))
	assert.True(t, ok)
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID2.String()))
	assert.True(t, ok)

	// 1. Remove schedule 1 from placement 1 (leaving rule 1 orphaned)
	mem.Schedules.Set(cmsPlacementSchedulesKey(placement1), []*entity.Schedule{}, time.Hour)
	mem.PruneOrphanedRules()

	// Verify rule 1 is gone, rule 2 remains
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID1.String()))
	assert.False(t, ok, "rule 1 should be pruned")
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID2.String()))
	assert.True(t, ok, "rule 2 should remain")

	// 2. Add rule 1 back to placement 2 (sharing rule 2)
	schedules2Updated := []*entity.Schedule{
		{BaseModel: entity.BaseModel{ID: uuid.New()}, DecisionRuleID: ruleID2},
		{BaseModel: entity.BaseModel{ID: uuid.New()}, DecisionRuleID: ruleID1},
	}
	mem.Schedules.Set(cmsPlacementSchedulesKey(placement2), schedules2Updated, time.Hour)
	mem.DecisionRule.Set(ruleDecisionCacheKey(ruleID1.String()), &entity.DecisionRule{}, time.Hour)
	mem.PruneOrphanedRules()

	// Both should exist now
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID1.String()))
	assert.True(t, ok)
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID2.String()))
	assert.True(t, ok)

	// 3. Clear placement 2 entirely
	mem.Schedules.Set(cmsPlacementSchedulesKey(placement2), []*entity.Schedule{}, time.Hour)
	mem.PruneOrphanedRules()

	// Both rules should be gone
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID1.String()))
	assert.False(t, ok)
	_, ok = mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID2.String()))
	assert.False(t, ok)
}

// ─────────────────────────────────────────────────────────────────────────────
// subscribeToUpdates tests
// ─────────────────────────────────────────────────────────────────────────────

// waitForSubDone blocks until the subscriber goroutine has initialised subDone.
// subscribeToUpdates sets subDone under the mutex at the very start, so this
// should return almost immediately.
func waitForSubDone(t *testing.T, svc *CMSDeliveryService) {
	t.Helper()
	require.Eventually(t, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		return svc.subDone != nil
	}, time.Second, 5*time.Millisecond)
}

// TestSubscribeToUpdates_SkipsMatchingVersion proves that when the local
// version hash already equals the incoming ping's hash, evaluate is never
// called — the message is dropped before the jitter delay even fires.
func TestSubscribeToUpdates_SkipsMatchingVersion(t *testing.T) {
	t.Parallel()

	msgCh := make(chan string, 1)
	evaluateCalled := make(chan string, 1) // receives placement name if called

	mem := newTestMemCache(t, "sub_skip_version")
	mem.VersionHashes.Set("hero", "abc123", time.Hour)

	occ := &mockOccurrenceRepo{
		listActiveByPlacementsAtFn: func(_ context.Context, names []string, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
			if len(names) > 0 {
				evaluateCalled <- names[0]
			}
			return nil, nil
		},
	}

	svc := NewCMSDeliveryService(
		&mockCacheRepo{subscribeFn: func(_ context.Context, _ string) (<-chan string, error) { return msgCh, nil }},
		occ, &mockDecisionRuleRepo{}, nil, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	go svc.subscribeToUpdates(t.Context())
	waitForSubDone(t, svc)

	payload, _ := json.Marshal(SyncPingMessage{PlacementName: "hero", VersionHash: "abc123"})
	msgCh <- string(payload)

	// The version check fires synchronously before jitter, so 150 ms is more
	// than enough time to confirm no evaluate was triggered.
	select {
	case name := <-evaluateCalled:
		t.Fatalf("evaluate should not have been called, but got placement %q", name)
	case <-time.After(150 * time.Millisecond):
		// expected: skip
	}
}

// TestSubscribeToUpdates_TriggersEvaluateForNewVersion proves that a
// SyncPingMessage whose version hash is not in the local cache causes
// evaluate to be called for the named placement.
func TestSubscribeToUpdates_TriggersEvaluateForNewVersion(t *testing.T) {
	t.Parallel()

	msgCh := make(chan string, 1)
	evaluateCalled := make(chan string, 1)

	mem := newTestMemCache(t, "sub_new_version")
	// "hero" has no cached version, so any hash must trigger a pull.

	occ := &mockOccurrenceRepo{
		listActiveByPlacementsAtFn: func(_ context.Context, names []string, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
			if len(names) > 0 {
				evaluateCalled <- names[0]
			}
			return nil, nil
		},
	}

	svc := NewCMSDeliveryService(
		&mockCacheRepo{subscribeFn: func(_ context.Context, _ string) (<-chan string, error) { return msgCh, nil }},
		occ, &mockDecisionRuleRepo{}, nil, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	go svc.subscribeToUpdates(t.Context())
	waitForSubDone(t, svc)

	payload, _ := json.Marshal(SyncPingMessage{PlacementName: "hero", VersionHash: "v2"})
	msgCh <- string(payload)

	// Jitter is at most 500 ms; allow 2 s for the evaluate to complete.
	select {
	case name := <-evaluateCalled:
		assert.Equal(t, "hero", name)
	case <-time.After(2 * time.Second):
		t.Fatal("evaluate was not called within 2 s after receiving a new-version ping")
	}
}

// TestSubscribeToUpdates_RawPayloadTriggersFull proves that an unparseable
// (non-JSON) message triggers a full evaluate across all placements rather
// than a targeted per-placement refresh.
func TestSubscribeToUpdates_RawPayloadTriggersFull(t *testing.T) {
	t.Parallel()

	msgCh := make(chan string, 1)
	fullEvaluateCalled := make(chan struct{}, 1)

	occ := &mockOccurrenceRepo{
		listActiveAtFn: func(_ context.Context, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
			fullEvaluateCalled <- struct{}{}
			return nil, nil
		},
	}

	svc := NewCMSDeliveryService(
		&mockCacheRepo{subscribeFn: func(_ context.Context, _ string) (<-chan string, error) { return msgCh, nil }},
		occ, &mockDecisionRuleRepo{}, nil, nil, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	go svc.subscribeToUpdates(t.Context())
	waitForSubDone(t, svc)

	msgCh <- "raw-ping-no-json"

	select {
	case <-fullEvaluateCalled:
		// expected: full evaluate (no placement filter)
	case <-time.After(2 * time.Second):
		t.Fatal("full evaluate was not called within 2 s after receiving a raw ping")
	}
}

// TestSubscribeToUpdates_SubscribeError proves that when the Redis Subscribe
// call itself fails, the subscriber goroutine exits cleanly (subDone is closed).
func TestSubscribeToUpdates_SubscribeError(t *testing.T) {
	t.Parallel()

	svc := NewCMSDeliveryService(
		&mockCacheRepo{subscribeFn: func(_ context.Context, _ string) (<-chan string, error) {
			return nil, errors.New("redis: connection refused")
		}},
		&mockOccurrenceRepo{}, &mockDecisionRuleRepo{}, nil, nil, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	go svc.subscribeToUpdates(t.Context())
	waitForSubDone(t, svc)

	select {
	case <-svc.subDone:
		// expected: goroutine exited after subscribe error
	case <-time.After(500 * time.Millisecond):
		t.Fatal("subscriber goroutine did not exit after Subscribe returned an error")
	}
}

// TestSubscribeToUpdates_ActivatePingDeletesBothCacheKeys proves that when a
// SyncPingMessage carries a non-empty DecisionRuleID (as published by the
// activate endpoint), both the rule and schedules cache entries are explicitly
// deleted before evaluate is called — ensuring no delivery pod can briefly
// observe a stale mirror between the delete and the re-evaluate.
func TestSubscribeToUpdates_ActivatePingDeletesBothCacheKeys(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	msgCh := make(chan string, 1)

	mem := newTestMemCache(t, "sub_activate_delete")
	// Pre-populate both entries that the activate ping is expected to clear.
	mem.DecisionRule.Set(ruleDecisionCacheKey(ruleID.String()), &entity.DecisionRule{}, time.Hour)
	mem.Schedules.Set(cmsPlacementSchedulesKey("hero"), []*entity.Schedule{}, time.Hour)

	evaluateCalled := make(chan struct{}, 1)
	occ := &mockOccurrenceRepo{
		listActiveByPlacementsAtFn: func(_ context.Context, _ []string, _ time.Time) ([]*entity.ScheduleOccurrence, error) {
			select {
			case evaluateCalled <- struct{}{}:
			default:
			}
			return nil, nil
		},
	}

	svc := NewCMSDeliveryService(
		&mockCacheRepo{subscribeFn: func(_ context.Context, _ string) (<-chan string, error) { return msgCh, nil }},
		occ, &mockDecisionRuleRepo{}, nil, mem, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	go svc.subscribeToUpdates(t.Context())
	waitForSubDone(t, svc)

	payload, _ := json.Marshal(SyncPingMessage{
		PlacementName:  "hero",
		VersionHash:    "", // empty → bypass version-hash short-circuit
		DecisionRuleID: ruleID.String(),
	})
	msgCh <- string(payload)

	// Explicit deletes fire after jitter (50–500 ms); allow 2 s total.
	require.Eventually(t, func() bool {
		_, rulePresent := mem.DecisionRule.Get(ruleDecisionCacheKey(ruleID.String()))
		_, schedPresent := mem.Schedules.Get(cmsPlacementSchedulesKey("hero"))
		return !rulePresent && !schedPresent
	}, 2*time.Second, 10*time.Millisecond, "expected both cache entries to be deleted after activate ping")

	// Confirm evaluate was also triggered after the explicit deletion.
	select {
	case <-evaluateCalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("evaluate was not called after the explicit cache deletion")
	}
}

// TestSubscribeToUpdates_ContextCancelExits proves that cancelling the parent
// context causes the subscriber goroutine to exit promptly without leaking.
func TestSubscribeToUpdates_ContextCancelExits(t *testing.T) {
	t.Parallel()

	// A never-closed channel simulates a live Redis subscription.
	msgCh := make(chan string)

	svc := NewCMSDeliveryService(
		&mockCacheRepo{subscribeFn: func(_ context.Context, _ string) (<-chan string, error) { return msgCh, nil }},
		&mockOccurrenceRepo{}, &mockDecisionRuleRepo{}, nil, nil, time.Hour, 0, nil, nil, CustomerProfileEnrichConfig{}, nil, nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go svc.subscribeToUpdates(ctx)
	waitForSubDone(t, svc)

	cancel()

	select {
	case <-svc.subDone:
		// expected: clean exit
	case <-time.After(500 * time.Millisecond):
		t.Fatal("subscriber goroutine did not exit within 500 ms of context cancellation")
	}
}
