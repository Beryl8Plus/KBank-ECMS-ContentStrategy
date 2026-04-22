package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	cmsruntimev1 "kbank-ecms/internal/grpc/pb/cms_runtime/v1"
)

func TestRuntimeGRPCServerEvaluateReturnsEmptyResponseWithoutUserAttrs(t *testing.T) {
	t.Parallel()

	server := NewRuntimeGRPCServer()
	attrID := uuid.New()
	decisionRuleID := uuid.New()

	req := buildEvaluateRequest(t, []*entity.Schedule{buildScheduleFixture(
		decisionRuleID,
		attrID,
		1.5,
		[]entity.Rule{
			buildRuleVariation("late", 2, 9, attrID, `"gold"`),
			buildRuleVariation("early", 1, 7, attrID, `"gold"`),
		},
	)}, map[string]json.RawMessage{
		attrID.String(): json.RawMessage(`"gold"`),
	})
	req.UserAttrs = nil

	resp, err := server.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

func TestRuntimeGRPCServerEvaluateReturnsHighestScoringMatchingVariation(t *testing.T) {
	t.Parallel()

	server := NewRuntimeGRPCServer()
	attrID := uuid.New()
	decisionRuleID := uuid.New()

	req := buildEvaluateRequest(t, []*entity.Schedule{buildScheduleFixture(
		decisionRuleID,
		attrID,
		2.5,
		[]entity.Rule{
			buildRuleVariation("late", 2, 9, attrID, `"gold"`),
			buildRuleVariation("early", 1, 7, attrID, `"gold"`),
		},
	)}, map[string]json.RawMessage{
		attrID.String(): json.RawMessage(`"gold"`),
	})

	resp, err := server.Evaluate(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, resp.Results, 1)
	assert.Equal(t, 9.0, resp.Results[0].Score)
	if assert.NotNil(t, resp.Results[0].Variation) {
		assert.Equal(t, "late", *resp.Results[0].Variation)
	}
	assert.NotEmpty(t, resp.Results[0].LogicHash)
	require.Len(t, resp.Results[0].Conditions, 1)
}

func TestRuntimeGRPCServerEvaluateReturnsEmptyWhenNoVariationMatches(t *testing.T) {
	t.Parallel()

	server := NewRuntimeGRPCServer()
	attrID := uuid.New()
	decisionRuleID := uuid.New()

	req := buildEvaluateRequest(t, []*entity.Schedule{buildScheduleFixture(
		decisionRuleID,
		attrID,
		2.5,
		[]entity.Rule{
			buildRuleVariation("matchable", 1, 10, attrID, `"gold"`),
		},
	)}, map[string]json.RawMessage{
		attrID.String(): json.RawMessage(`"silver"`),
	})

	resp, err := server.Evaluate(context.Background(), req)
	require.NoError(t, err)

	assert.Empty(t, resp.Results)
}

func buildEvaluateRequest(
	t *testing.T,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) *cmsruntimev1.EvaluateRequest {
	t.Helper()

	schedulesJSON, err := json.Marshal(schedules)
	require.NoError(t, err)

	// Build native proto user attrs map (map[string][]byte).
	protoAttrs := make(map[string][]byte, len(userAttrs))
	for k, v := range userAttrs {
		protoAttrs[k] = []byte(v)
	}

	return &cmsruntimev1.EvaluateRequest{
		PlacementName: "hero",
		SchedulesJson: schedulesJSON,
		UserAttrs:     protoAttrs,
	}
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
