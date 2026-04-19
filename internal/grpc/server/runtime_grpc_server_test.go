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
	domainservice "kbank-ecms/internal/domain/service"
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
	req.UserAttrsJson = nil

	resp, err := server.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.Empty(t, resp.LogicEntriesJson)
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

	results := decodeEvaluateResponse(t, resp)
	require.Len(t, results, 1)
	assert.Equal(t, 9.0, results[0].Score)
	if assert.NotNil(t, results[0].Variation) {
		assert.Equal(t, "late", *results[0].Variation)
	}
	assert.NotEmpty(t, results[0].LogicHash)
	require.Len(t, results[0].Conditions, 1)
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

	results := decodeEvaluateResponse(t, resp)
	assert.Empty(t, results)
}

func buildEvaluateRequest(
	t *testing.T,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) *cmsruntimev1.EvaluateRequest {
	t.Helper()

	schedulesJSON, err := json.Marshal(schedules)
	require.NoError(t, err)

	var userAttrsJSON []byte
	if len(userAttrs) > 0 {
		userAttrsJSON, err = json.Marshal(userAttrs)
		require.NoError(t, err)
	}

	return &cmsruntimev1.EvaluateRequest{
		PlacementName: "hero",
		SchedulesJson: schedulesJSON,
		UserAttrsJson: userAttrsJSON,
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

func decodeEvaluateResponse(t *testing.T, resp *cmsruntimev1.EvaluateResponse) []domainservice.ContentResult {
	t.Helper()

	var entries []domainservice.ContentResult
	require.NoError(t, json.Unmarshal(resp.LogicEntriesJson, &entries))
	return entries
}
