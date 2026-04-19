package client

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	grpcserver "kbank-ecms/internal/grpc/server"
)

const bufConnSize = 1024 * 1024

func TestRuntimeGRPCClientEvaluateReturnsNilWithoutUserAttrs(t *testing.T) {
	t.Parallel()

	client := newBufconnRuntimeClient(t)
	attrID := uuid.New()
	decisionRuleID := uuid.New()

	results, err := client.Evaluate(
		context.Background(),
		"hero",
		[]*entity.Schedule{buildScheduleFixture(
			decisionRuleID,
			attrID,
			1.5,
			[]entity.Rule{
				buildRuleVariation("late", 2, 9, attrID, `"gold"`),
				buildRuleVariation("early", 1, 7, attrID, `"gold"`),
			},
		)},
		nil,
	)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestRuntimeGRPCClientEvaluateReturnsResolvedRankedResult(t *testing.T) {
	t.Parallel()

	client := newBufconnRuntimeClient(t)
	attrID := uuid.New()
	decisionRuleID := uuid.New()
	userAttrs := map[string]json.RawMessage{
		attrID.String(): json.RawMessage(`"gold"`),
	}

	results, err := client.Evaluate(
		context.Background(),
		"hero",
		[]*entity.Schedule{buildScheduleFixture(
			decisionRuleID,
			attrID,
			2.5,
			[]entity.Rule{
				buildRuleVariation("late", 2, 9, attrID, `"gold"`),
				buildRuleVariation("early", 1, 7, attrID, `"gold"`),
			},
		)},
		userAttrs,
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 9.0, results[0].Score)
	if assert.NotNil(t, results[0].Variation) {
		assert.Equal(t, "late", *results[0].Variation)
	}
	assert.NotEmpty(t, results[0].LogicHash)
	require.Len(t, results[0].Conditions, 1)
}

func newBufconnRuntimeClient(t *testing.T) *RuntimeGRPCClient {
	t.Helper()

	listener := bufconn.Listen(bufConnSize)
	grpcSrv := grpc.NewServer()
	grpcserver.Register(grpcSrv)

	go func() {
		_ = grpcSrv.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcSrv.Stop()
		require.NoError(t, listener.Close())
	})

	client, err := newRuntimeGRPCClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	return client
}

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
