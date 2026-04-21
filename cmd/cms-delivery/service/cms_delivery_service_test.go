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
	domainservice "kbank-ecms/internal/domain/service"
	"kbank-ecms/internal/infrastructure/cache"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers: constructor helpers to reduce boilerplate in tests
// ─────────────────────────────────────────────────────────────────────────────

// newSvcCacheOnly builds a CMSDeliveryService with only a cache repo (no gRPC).
func newSvcCacheOnly(cacheRepo *mockCacheRepo) *CMSDeliveryService {
	return NewCMSDeliveryService(cacheRepo, &mockScheduleRepo{}, &mockDecisionRuleRepo{}, nil, nil, time.Hour, 0)
}

// newSvcWithFallback builds a CMSDeliveryService with cache, schedule repo, and evaluator.
func newSvcWithFallback(
	cacheRepo *mockCacheRepo,
	schedRepo *mockScheduleRepo,
	eval domainservice.RuntimeEvaluator,
) *CMSDeliveryService {
	return NewCMSDeliveryService(cacheRepo, schedRepo, &mockDecisionRuleRepo{}, eval, nil, time.Hour, 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// Logic-cache miss + gRPC fallback tests
// ─────────────────────────────────────────────────────────────────────────────

// TestGetPersonalizedContent_LogicCacheMiss_WithGRPCFallback verifies that a
// placement logic cache miss calls RuntimeEvaluator.Evaluate, then caches the
// user evaluation result and the personalized placement result.
func TestGetPersonalizedContent_LogicCacheMiss_WithGRPCFallback(t *testing.T) {
	t.Parallel()

	logicEntry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/content/hero",
		RuleSetType:    "MASS",
		Score:          0.8,
		LogicHash:      "hero-hash",
		LogicEval:      true, // server evaluated this entry as matching
		Conditions:     []dto.LogicCondition{},
	}

	// Pre-populate the schedule cache so filtered["hero"] has 1 entry.
	sched := &entity.Schedule{Placement: &entity.Placement{Name: "hero", MaxResults: 5}}
	schedJSON, _ := json.Marshal([]*entity.Schedule{sched})
	stored := map[string]string{
		cmsPlacementSchedulesKey("hero"): string(schedJSON),
	}

	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if value, ok := stored[key]; ok {
					return value, nil
				}
				return "", errors.New("miss")
			},
			setFn: func(_ context.Context, key, value string, _ time.Duration) error {
				stored[key] = value
				return nil
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, name string, schedules []*entity.Schedule, userAttrs map[string]json.RawMessage) ([]dto.ContentResult, error) {
				assert.Equal(t, "hero", name)
				require.Len(t, schedules, 1)
				assert.Empty(t, userAttrs) // service passes resolved (empty) user attrs
				return []dto.ContentResult{logicEntry}, nil
			},
		},
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(),
		"cis1",
		"user1",
		[]string{"hero"},
		map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/content/hero", result[0].ContentPath)
	assert.Equal(t, "true", stored[cmsUserEvalKey("user1", logicEntry.LogicHash)])
	assert.NotEmpty(t, stored[cmsPersonalizedPlacementKey("cis1", "hero")])
}

// TestGetPersonalizedContent_LogicCacheMiss_GRPCFails verifies graceful
// degradation: gRPC failure returns an empty result without propagating the error.
func TestGetPersonalizedContent_LogicCacheMiss_GRPCFails(t *testing.T) {
	t.Parallel()

	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("miss")
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return nil, errors.New("rpc error: unavailable")
			},
		},
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(),
		"cis1",
		"user1",
		[]string{"hero"},
		map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// FlushCache tests
// ─────────────────────────────────────────────────────────────────────────────

// TestCMSDeliveryService_FlushCache_Selective verifies that FlushCache calls
// Delete for each named placement (schedules key + placement key) and does not
// call FlushDB. Rule caches are also flushed when schedule data is present.
func TestCMSDeliveryService_FlushCache_Selective(t *testing.T) {
	t.Parallel()

	deletedKeys := []string{}
	flushDBCalled := false
	mem := cache.NewCacheMemory[any]("flush_cache_selective_test", 0.95)
	t.Cleanup(mem.Stop)
	mem.Set(cmsPlacementKey("hero"), "hero", time.Hour)
	mem.Set(cmsPlacementKey("sidebar"), "sidebar", time.Hour)

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

	err := svc.FlushCache(context.Background(), []string{"hero", "sidebar"}, true)

	require.NoError(t, err)
	assert.False(t, flushDBCalled)
	// UC3: each placement flushes schedules key + placement key = 2 × 2 = 4 deletes.
	// (no rule keys because schedule cache is empty → no rule IDs to resolve)
	assert.Len(t, deletedKeys, 4)
	assert.Contains(t, deletedKeys, cmsPlacementSchedulesKey("hero"))
	assert.Contains(t, deletedKeys, cmsPlacementKey("hero"))
	assert.Contains(t, deletedKeys, cmsPlacementSchedulesKey("sidebar"))
	assert.Contains(t, deletedKeys, cmsPlacementKey("sidebar"))
	_, ok := mem.Get(context.Background(), cmsPlacementKey("hero"))
	assert.False(t, ok)
	_, ok = mem.Get(context.Background(), cmsPlacementKey("sidebar"))
	assert.False(t, ok)
}

// TestCMSDeliveryService_FlushCache_All verifies that FlushCache with nil/empty
// names calls FlushDB instead of Delete.
func TestCMSDeliveryService_FlushCache_All(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	flushDBCalled := false
	mem := cache.NewCacheMemory[any]("flush_cache_all_test", 0.95)
	t.Cleanup(mem.Stop)
	mem.Set(cmsPlacementKey("hero"), "hero", time.Hour)

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

	// nil placements
	require.NoError(t, svc.FlushCache(context.Background(), nil, false))
	assert.True(t, flushDBCalled)
	assert.False(t, deleteCalled)
	_, ok := mem.Get(context.Background(), cmsPlacementKey("hero"))
	assert.False(t, ok)

	// reset and test empty slice
	flushDBCalled = false
	mem.Set(cmsPlacementKey("hero"), "hero", time.Hour)
	require.NoError(t, svc.FlushCache(context.Background(), []string{}, false))
	assert.True(t, flushDBCalled)
	_, ok = mem.Get(context.Background(), cmsPlacementKey("hero"))
	assert.False(t, ok)
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
func (m *mockCacheRepo) HGet(ctx context.Context, key, field string) (string, error) { return "", nil }
func (m *mockCacheRepo) HSet(ctx context.Context, key, field, val string) error      { return nil }
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

// mockScheduleRepo is a minimal mock for domainrepo.ScheduleRepository.
type mockScheduleRepo struct {
	listActiveFn func(ctx context.Context, at time.Time) ([]*entity.Schedule, error)
}

func (m *mockScheduleRepo) ListActiveSchedulesInWindow(ctx context.Context, at time.Time) ([]*entity.Schedule, error) {
	if m.listActiveFn != nil {
		return m.listActiveFn(ctx, at)
	}
	return nil, nil
}

func (m *mockScheduleRepo) CheckScheduleOverlap(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) CreateSchedule(_ context.Context, _ *entity.Schedule) error { return nil }
func (m *mockScheduleRepo) GetScheduleByID(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) ListSchedules(_ context.Context) ([]*entity.Schedule, error) {
	return nil, nil
}
func (m *mockScheduleRepo) ListSchedulesPaginated(_ context.Context, _, _ int) ([]*entity.Schedule, int64, error) {
	return nil, 0, nil
}
func (m *mockScheduleRepo) UpdateSchedule(_ context.Context, _ *entity.Schedule) error { return nil }
func (m *mockScheduleRepo) DeleteSchedule(_ context.Context, _ uuid.UUID) error        { return nil }

// mockDecisionRuleRepo is a minimal mock for domainrepo.DecisionRuleRepository.
type mockDecisionRuleRepo struct {
	getFn func(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error)
}

func (m *mockDecisionRuleRepo) GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error) {
	if m.getFn != nil {
		return m.getFn(ctx, scheduleID)
	}
	return nil, nil
}

// mockRuntimeEvaluator is a minimal mock for domainservice.RuntimeEvaluator.
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
var _ domainservice.RuntimeEvaluator = (*mockRuntimeEvaluator)(nil)

// ─────────────────────────────────────────────────────────────────────────────
// GetPersonalizedContent tests
// ─────────────────────────────────────────────────────────────────────────────

// TestGetPersonalizedContent_PersonalizedCacheHit verifies that a warm
// cms:placement:{cisID}:{name} cache is returned without any evaluation.
func TestGetPersonalizedContent_PersonalizedCacheHit(t *testing.T) {
	t.Parallel()

	cached := []dto.ContentResult{{ContentPath: "/hero", Score: 0.9}}
	data, _ := json.Marshal(cached)

	svc := newSvcCacheOnly(&mockCacheRepo{
		getFn: func(_ context.Context, key string) (string, error) {
			if key == cmsPersonalizedPlacementKey("cis1", "hero") {
				return string(data), nil
			}
			return "", errors.New("not found")
		},
	})

	result, err := svc.GetPersonalizedContent(context.Background(), "cis1", "user1", []string{"hero"}, nil)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/hero", result[0].ContentPath)
}

// TestGetPersonalizedContent_UserEvalCacheHit_True verifies that a cached
// "true" user-eval result skips live evaluation and includes the entry.
func TestGetPersonalizedContent_UserEvalCacheHit_True(t *testing.T) {
	t.Parallel()

	logicHash := "abc123"
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/product",
		RuleSetType:    "Mass",
		Score:          0.8,
		LogicHash:      logicHash,
		Conditions:     []dto.LogicCondition{},
	}

	// gRPC returns entry; per-user eval cache is pre-populated as "true".
	stored := map[string]string{
		cmsUserEvalKey("user1", logicHash): "true",
	}
	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
			setFn: func(_ context.Context, key, val string, _ time.Duration) error {
				stored[key] = val
				return nil
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return []dto.ContentResult{entry}, nil
			},
		},
	)

	result, err := svc.GetPersonalizedContent(context.Background(), "cis1", "user1", []string{"shopping"}, nil)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/product", result[0].ContentPath)
}

// TestGetPersonalizedContent_UserEvalCacheHit_False verifies that a cached
// "false" user-eval result excludes the entry.
func TestGetPersonalizedContent_UserEvalCacheHit_False(t *testing.T) {
	t.Parallel()

	logicHash := "def456"
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/product",
		LogicHash:      logicHash,
		LogicEval:      true, // LogicEval is true but cached eval says "false" — cached result wins
		Conditions:     []dto.LogicCondition{},
	}

	// gRPC returns entry; per-user eval cache is pre-populated as "false".
	stored := map[string]string{
		cmsUserEvalKey("user1", logicHash): "false",
	}
	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return []dto.ContentResult{entry}, nil
			},
		},
	)

	result, err := svc.GetPersonalizedContent(context.Background(), "cis1", "user1", []string{"shop"}, nil)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetPersonalizedContent_LiveEval_PassAndCache verifies that a user-eval
// cache miss triggers live evaluation via entry.LogicEval; the result is stored
// and the passing entry is returned.
func TestGetPersonalizedContent_LiveEval_PassAndCache(t *testing.T) {
	t.Parallel()

	logicHash := "live-pass-hash"
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/offer",
		Score:          1.0,
		LogicHash:      logicHash,
		LogicEval:      true, // server evaluated this entry as matching
		Conditions:     []dto.LogicCondition{},
	}

	stored := map[string]string{}
	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
			setFn: func(_ context.Context, key, val string, _ time.Duration) error {
				stored[key] = val
				return nil
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return []dto.ContentResult{entry}, nil
			},
		},
	)

	userAttrs := map[string]json.RawMessage{
		uuid.New().String(): json.RawMessage(`"premium"`),
	}

	result, err := svc.GetPersonalizedContent(context.Background(), "cis2", "user99", []string{"deals"}, userAttrs)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/offer", result[0].ContentPath)

	// User eval result should be cached under the preserved logicHash.
	assert.Equal(t, "true", stored[cmsUserEvalKey("user99", logicHash)])
	// Personalised placement cache should be written.
	assert.NotEmpty(t, stored[cmsPersonalizedPlacementKey("cis2", "deals")])
}

// TestGetPersonalizedContent_LoadsUserAttrsFromRedisByCISID verifies that the
// delivery service resolves a cis_id:{cisID} JSON payload from Redis and passes
// the merged attributes to the gRPC evaluator.
func TestGetPersonalizedContent_LoadsUserAttrsFromRedisByCISID(t *testing.T) {
	t.Parallel()

	condAttrID := uuid.New()
	logicHash := "cis-redis-hash"
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/wealth/vip",
		Score:          9.5,
		LogicHash:      logicHash,
		LogicEval:      true, // server evaluated this entry as matching
	}

	userAttrsData, _ := json.Marshal(map[string]interface{}{
		condAttrID.String(): "balanced",
	})

	stored := map[string]string{
		cmsUserAttrsKey("cis42"): string(userAttrsData),
	}

	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
			setFn: func(_ context.Context, key, val string, _ time.Duration) error {
				stored[key] = val
				return nil
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, userAttrs map[string]json.RawMessage) ([]dto.ContentResult, error) {
				// Verify the user attr from Redis was resolved and passed to the evaluator.
				_, ok := userAttrs[condAttrID.String()]
				assert.True(t, ok, "expected resolved user attr to be passed to evaluator")
				return []dto.ContentResult{entry}, nil
			},
		},
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis42", "user88", []string{"wealth-banner"}, nil,
	)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "/wealth/vip", result[0].ContentPath)
	assert.Equal(t, "true", stored[cmsUserEvalKey("user88", logicHash)])
	assert.NotEmpty(t, stored[cmsPersonalizedPlacementKey("cis42", "wealth-banner")])
}

// TestGetPersonalizedContent_EvalFalse_ExcludedAndCached verifies that when
// the evaluator returns LogicEval=false the entry is excluded and the result
// is cached as "false" under the user eval key.
func TestGetPersonalizedContent_EvalFalse_ExcludedAndCached(t *testing.T) {
	t.Parallel()

	logicHash := "eval-false-hash"
	entry := dto.ContentResult{
		DecisionRuleId: uuid.New().String(),
		ContentPath:    "/restricted",
		LogicHash:      logicHash,
		LogicEval:      false, // evaluator determined this entry does not match
	}

	stored := map[string]string{}
	svc := newSvcWithFallback(
		&mockCacheRepo{
			getFn: func(_ context.Context, key string) (string, error) {
				if v, ok := stored[key]; ok {
					return v, nil
				}
				return "", errors.New("miss")
			},
			setFn: func(_ context.Context, key, val string, _ time.Duration) error {
				stored[key] = val
				return nil
			},
		},
		&mockScheduleRepo{},
		&mockRuntimeEvaluator{
			evaluateFn: func(_ context.Context, _ string, _ []*entity.Schedule, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
				return []dto.ContentResult{entry}, nil
			},
		},
	)

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis3", "user42", []string{"vip"}, map[string]json.RawMessage{},
	)

	require.NoError(t, err)
	assert.Empty(t, result)
	// The false result should be cached under the logic hash.
	assert.Equal(t, "false", stored[cmsUserEvalKey("user42", logicHash)])
}

// TestGetPersonalizedContent_NoLogicCache returns empty when the logic key is absent.
func TestGetPersonalizedContent_NoLogicCache(t *testing.T) {
	t.Parallel()

	svc := newSvcCacheOnly(&mockCacheRepo{
		getFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("miss")
		},
	})

	result, err := svc.GetPersonalizedContent(
		context.Background(), "cis1", "user1", []string{"unknown-placement"}, nil,
	)

	require.NoError(t, err)
	assert.Empty(t, result)
}
