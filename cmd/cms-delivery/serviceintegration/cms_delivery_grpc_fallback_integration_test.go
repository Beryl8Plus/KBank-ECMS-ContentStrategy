package serviceintegration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"gorm.io/datatypes"

	deliveryservice "kbank-ecms/cmd/cms-delivery/service"
	grpcserver "kbank-ecms/cmd/cms-runtime/testserver"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
	grpcclient "kbank-ecms/internal/grpc/client"
)

func TestCMSDeliveryServiceGRPCFallbackEndToEnd(t *testing.T) {
	t.Parallel()

	addr := startRuntimeServer(t)
	evaluator, err := grpcclient.NewRuntimeGRPCClient(addr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, evaluator.Close())
	})

	attrID := uuid.New()
	cacheRepo := newMemoryCacheRepo()

	// Pre-populate the schedule cache so GetPersonalizedContent can resolve schedules.
	sched := buildScheduleFixture(
		uuid.New(),
		attrID,
		2.5,
		[]entity.Rule{
			buildRuleVariation("late", 2, 9, attrID, `"silver"`),
			buildRuleVariation("early", 1, 7, attrID, `"gold"`),
		},
	)
	schedJSON, err := json.Marshal([]*entity.Schedule{sched})
	require.NoError(t, err)
	require.NoError(t, cacheRepo.Set(context.Background(), placementSchedulesKey("hero"), string(schedJSON), time.Hour))

	svc := deliveryservice.NewCMSDeliveryService(
		cacheRepo,
		&occurrenceRepoStub{}, // non-nil guard enables gRPC fallback; not called directly
		&decisionRuleRepoStub{},
		evaluator,
		nil,
		time.Hour,
		0,
	)

	userAttrs := map[string]json.RawMessage{
		attrID.String(): json.RawMessage(`"gold"`),
	}

	results, err := svc.GetPersonalizedContent(
		context.Background(),
		"cis-e2e",
		"user-e2e",
		[]string{"hero"},
		userAttrs,
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/hero", results[0].ContentPath)
	assert.Equal(t, 7.0, results[0].Score)
	if assert.NotNil(t, results[0].Variation) {
		assert.Equal(t, "early", *results[0].Variation)
	}

	// The gRPC evaluator embeds LogicHash into each ContentResult.
	// Use it to verify that the per-user eval result was written to cache.
	require.NotEmpty(t, results[0].LogicHash)
	assert.Equal(t, "true", cacheRepo.mustGet(userEvalKey("user-e2e", results[0].LogicHash)))
	assert.NotEmpty(t, cacheRepo.mustGet(personalizedPlacementKey("cis-e2e", "hero")))
}

func startRuntimeServer(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcSrv := grpc.NewServer()
	grpcserver.Register(grpcSrv)

	go func() {
		_ = grpcSrv.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcSrv.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

type memoryCacheRepo struct {
	mu    sync.RWMutex
	store map[string]string
}

func newMemoryCacheRepo() *memoryCacheRepo {
	return &memoryCacheRepo{store: make(map[string]string)}
}

func (m *memoryCacheRepo) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.store[key]
	if !ok {
		return "", errors.New("miss")
	}
	return value, nil
}

func (m *memoryCacheRepo) Set(_ context.Context, key string, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
	return nil
}

func (m *memoryCacheRepo) HGet(_ context.Context, _, _ string) (string, error) { return "", nil }
func (m *memoryCacheRepo) HSet(_ context.Context, _, _, _ string) error        { return nil }

func (m *memoryCacheRepo) FlushDB(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]string)
	return nil
}

func (m *memoryCacheRepo) GetSet(ctx context.Context, key string, expiration time.Duration, loader func(context.Context) (string, error)) (string, error) {
	if value, err := m.Get(ctx, key); err == nil {
		return value, nil
	}
	value, err := loader(ctx)
	if err != nil {
		return "", err
	}
	if err := m.Set(ctx, key, value, expiration); err != nil {
		return "", err
	}
	return value, nil
}

func (m *memoryCacheRepo) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}

func (m *memoryCacheRepo) mustGet(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store[key]
}

var _ domainrepo.RedisCacheRepository = (*memoryCacheRepo)(nil)

type occurrenceRepoStub struct{}

func (s *occurrenceRepoStub) ListActiveAt(context.Context, time.Time) ([]*entity.ScheduleOccurrence, error) {
	return nil, nil
}
func (s *occurrenceRepoStub) UpsertOccurrences(context.Context, []*entity.ScheduleOccurrence) error {
	return nil
}
func (s *occurrenceRepoStub) DeleteFutureByScheduleID(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (s *occurrenceRepoStub) DeletePastOccurrences(context.Context, time.Time) error { return nil }
func (s *occurrenceRepoStub) ListByScheduleID(context.Context, uuid.UUID, int, int) ([]*entity.ScheduleOccurrence, int64, error) {
	return nil, 0, nil
}

var _ domainrepo.ScheduleOccurrenceRepository = (*occurrenceRepoStub)(nil)

type decisionRuleRepoStub struct{}

func (s *decisionRuleRepoStub) GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error) {
	return nil, nil
}

var _ domainrepo.DecisionRuleRepository = (*decisionRuleRepoStub)(nil)

func buildScheduleFixture(
	decisionRuleID uuid.UUID,
	attrID uuid.UUID,
	ruleScore float64,
	variations []entity.Rule,
) *entity.Schedule {
	now := time.Date(2026, time.April, 18, 12, 0, 0, 0, time.UTC)
	conditionID := uuid.New()

	rule := &entity.DecisionRule{
		BaseModel:   entity.BaseModel{ID: decisionRuleID},
		Type:        enums.DecisionTypeMass,
		ContentPath: "/hero",
		Score:       ruleScore,
		RuleConditions: []entity.RuleCondition{{
			BaseModel:         entity.BaseModel{ID: conditionID},
			Sequence:          1,
			DecisionRuleID:    decisionRuleID,
			AttributeID:       attrID,
			Attribute:         &entity.Attribute{BaseModel: entity.BaseModel{ID: attrID}, DataType: enums.AttributeDataTypeText},
			LogicalOperator:   enums.LogicalOperatorEQ,
			ConnectorOperator: enums.ConnectorOperatorAND,
		}},
		Rules: variations,
	}

	return &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: decisionRuleID,
		DecisionRule:   rule,
		Placement:      &entity.Placement{Name: "hero", MaxResults: 10},
		EffectiveFrom:  now,
		EffectiveUntil: now.Add(24 * time.Hour),
	}
}

func buildRuleVariation(name string, orderNo int, score float64, attrID uuid.UUID, value string) entity.Rule {
	ruleID := uuid.New()
	return entity.Rule{
		BaseModel:      entity.BaseModel{ID: ruleID},
		DecisionRuleID: uuid.Nil,
		VariationName:  name,
		Score:          score,
		OrderNo:        orderNo,
		RuleAttributes: []entity.RuleAttribute{{
			BaseModel:   entity.BaseModel{ID: uuid.New()},
			RuleID:      ruleID,
			AttributeID: attrID,
			Value:       datatypes.JSON(value),
		}},
	}
}

func placementSchedulesKey(name string) string {
	return fmt.Sprintf("cms:placement:%s:schedules", name)
}

func personalizedPlacementKey(cisID, name string) string {
	return "cms:placement:" + cisID + ":" + name
}

func userEvalKey(userID, hash string) string {
	return "cms:eval:user:" + userID + ":logic:" + hash
}
