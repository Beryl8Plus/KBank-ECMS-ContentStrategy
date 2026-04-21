package evaluator

import (
	"encoding/json"
	"testing"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustJSON(v string) json.RawMessage { return json.RawMessage(v) }

// buildSimpleRule builds a DecisionRule with one RuleCondition and one Rule variation.
//
//	attrID        — attribute UUID used in both condition and variation
//	expectedValue — value stored in RuleAttribute (the "expected" threshold)
//	op            — logical operator applied in the condition
func buildSimpleRule(attrID uuid.UUID, expectedValue string, op enums.LogicalOperator) entity.DecisionRule {
	condID := uuid.New()
	ruleID := uuid.New()

	cond := entity.RuleCondition{
		BaseModel:         entity.BaseModel{ID: condID},
		AttributeID:       attrID,
		Sequence:          1,
		LogicalOperator:   op,
		ConnectorOperator: enums.ConnectorOperatorAND,
		Attribute: &entity.Attribute{
			DataType: enums.AttributeDataTypeText,
		},
	}

	variation := entity.Rule{
		BaseModel:     entity.BaseModel{ID: ruleID},
		VariationName: "varA",
		Score:         10,
		OrderNo:       1,
		RuleAttributes: []entity.RuleAttribute{
			{
				AttributeID: attrID,
				Value:       datatypes.JSON(expectedValue),
			},
		},
	}

	return entity.DecisionRule{
		Score:          1.0,
		RuleConditions: []entity.RuleCondition{cond},
		Rules:          []entity.Rule{variation},
	}
}

// ---------------------------------------------------------------------------
// TestEvaluateRuleScore_NilUserAttrs — nil userAttrs treats conditions as non-match
// ---------------------------------------------------------------------------

func TestEvaluateRuleScore_NilUserAttrs(t *testing.T) {
	attrID := uuid.New()

	t.Run("NoConditions_ReturnsDefaultScore", func(t *testing.T) {
		rule := entity.DecisionRule{Score: 5.0}
		v, score, err := EvaluateRuleScore(rule, nil)
		require.NoError(t, err)
		assert.Nil(t, v)
		assert.Equal(t, 5.0, score)
	})

	t.Run("WithConditions_NilUserAttrs_ReturnsDefaultScore", func(t *testing.T) {
		rule := buildSimpleRule(attrID, `"gold"`, enums.LogicalOperatorEQ)
		v, score, err := EvaluateRuleScore(rule, nil)
		require.NoError(t, err)
		assert.Nil(t, v)
		assert.Equal(t, rule.Score, score) // no match → fallback
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateRuleScore_WithUserAttrs — non-nil userAttrs provides actual values
// ---------------------------------------------------------------------------

func TestEvaluateRuleScore_WithUserAttrs(t *testing.T) {
	attrID := uuid.New()

	t.Run("Pass_UserAttrMatchesExpected", func(t *testing.T) {
		rule := buildSimpleRule(attrID, `"gold"`, enums.LogicalOperatorEQ)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"gold"`),
		}
		v, score, err := EvaluateRuleScore(rule, userAttrs)
		require.NoError(t, err)
		require.NotNil(t, v)
		assert.Equal(t, "varA", *v)
		assert.Equal(t, float64(10), score)
	})

	t.Run("Fail_UserAttrDoesNotMatchExpected", func(t *testing.T) {
		rule := buildSimpleRule(attrID, `"gold"`, enums.LogicalOperatorEQ)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"silver"`),
		}
		v, score, err := EvaluateRuleScore(rule, userAttrs)
		require.NoError(t, err)
		assert.Nil(t, v)
		assert.Equal(t, rule.Score, score)
	})

	t.Run("MissingAttr_TreatedAsNonMatch", func(t *testing.T) {
		rule := buildSimpleRule(attrID, `"gold"`, enums.LogicalOperatorEQ)
		// userAttrs provided but does not contain attrID
		userAttrs := map[string]json.RawMessage{
			uuid.New().String(): mustJSON(`"gold"`),
		}
		v, score, err := EvaluateRuleScore(rule, userAttrs)
		require.NoError(t, err)
		assert.Nil(t, v)
		assert.Equal(t, rule.Score, score)
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateLogicConditions_UnifiedPath — regression tests
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_UnifiedPath(t *testing.T) {
	attrID := uuid.New()

	makeLogicCond := func(attrIDStr, expectedVal, logicalOp, dataType string) dto.LogicCondition {
		return dto.LogicCondition{
			ConditionID:       uuid.New().String(),
			AttributeID:       attrIDStr,
			DataType:          dataType,
			LogicalOperator:   logicalOp,
			ConnectorOperator: string(enums.ConnectorOperatorAND),
			Sequence:          1,
			ExpectedValue:     mustJSON(expectedVal),
		}
	}

	t.Run("EmptyConditions_ReturnsTrue", func(t *testing.T) {
		ok, err := EvaluateLogicConditions(nil, map[string]json.RawMessage{})
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Pass_SingleTextEQ", func(t *testing.T) {
		cond := makeLogicCond(attrID.String(), `"gold"`, string(enums.LogicalOperatorEQ), string(enums.AttributeDataTypeText))
		userAttrs := map[string]json.RawMessage{attrID.String(): mustJSON(`"gold"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Fail_TextEQ_WrongValue", func(t *testing.T) {
		cond := makeLogicCond(attrID.String(), `"gold"`, string(enums.LogicalOperatorEQ), string(enums.AttributeDataTypeText))
		userAttrs := map[string]json.RawMessage{attrID.String(): mustJSON(`"silver"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("MissingAttr_ReturnsFalse", func(t *testing.T) {
		cond := makeLogicCond(attrID.String(), `"gold"`, string(enums.LogicalOperatorEQ), string(enums.AttributeDataTypeText))
		userAttrs := map[string]json.RawMessage{} // attr not present
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("TwoConditions_AND_BothPass", func(t *testing.T) {
		attr2 := uuid.New()
		cond1 := makeLogicCond(attrID.String(), `"gold"`, string(enums.LogicalOperatorEQ), string(enums.AttributeDataTypeText))
		cond1.Sequence = 1
		cond1.ConnectorOperator = string(enums.ConnectorOperatorAND)

		cond2 := makeLogicCond(attr2.String(), `42`, string(enums.LogicalOperatorGTE), string(enums.AttributeDataTypeNumber))
		cond2.Sequence = 2
		cond2.ConnectorOperator = string(enums.ConnectorOperatorAND)

		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"gold"`),
			attr2.String():  mustJSON(`50`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond1, cond2}, userAttrs)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("TwoConditions_AND_OneFails", func(t *testing.T) {
		attr2 := uuid.New()
		cond1 := makeLogicCond(attrID.String(), `"gold"`, string(enums.LogicalOperatorEQ), string(enums.AttributeDataTypeText))
		cond1.Sequence = 1
		cond2 := makeLogicCond(attr2.String(), `100`, string(enums.LogicalOperatorGTE), string(enums.AttributeDataTypeNumber))
		cond2.Sequence = 2
		cond2.ConnectorOperator = string(enums.ConnectorOperatorAND)

		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"gold"`),
			attr2.String():  mustJSON(`50`), // below threshold
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond1, cond2}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateLogicConditions_NilOrNullExpectedValue
// Regression: a nil or JSON-null ExpectedValue must not bypass the !ok guard
// in evaluateSingleCondition by placing a nil/zero value in expectedValues.
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_NilOrNullExpectedValue(t *testing.T) {
	attrID := uuid.New()

	makeCondWithExpected := func(ev json.RawMessage) dto.LogicCondition {
		return dto.LogicCondition{
			ConditionID:       uuid.New().String(),
			AttributeID:       attrID.String(),
			DataType:          string(enums.AttributeDataTypeText),
			LogicalOperator:   string(enums.LogicalOperatorEQ),
			ConnectorOperator: string(enums.ConnectorOperatorAND),
			Sequence:          1,
			ExpectedValue:     ev,
		}
	}

	userAttrs := map[string]json.RawMessage{
		attrID.String(): mustJSON(`"gold"`),
	}

	t.Run("NilExpectedValue_ReturnsFalse", func(t *testing.T) {
		// ExpectedValue not stamped from rule_attribute.value (nil) — must be non-match.
		cond := makeCondWithExpected(nil)
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("JsonNullExpectedValue_ReturnsFalse", func(t *testing.T) {
		// After JSON round-trip through Redis, nil becomes json.RawMessage("null") — must also be non-match.
		cond := makeCondWithExpected(json.RawMessage("null"))
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
