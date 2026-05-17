package evaluator

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustJSON(v string) json.RawMessage { return json.RawMessage(v) }

func buildSimpleLogicCondition(attrID uuid.UUID, expectedValue string, op enums.LogicalOperator) dto.LogicCondition {
	return dto.LogicCondition{
		ConditionID:     uuid.New().String(),
		AttributeID:     attrID.String(),
		DataType:        string(enums.AttributeDataTypeText),
		LogicalOperator: string(op),
		Sequence:        1,
		ExpectedValue:   mustJSON(expectedValue),
	}
}

// ---------------------------------------------------------------------------
// TestEvaluateLogicConditions_NilUserAttrs - nil userAttrs treats conditions as non-match
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_NilUserAttrs(t *testing.T) {
	attrID := uuid.New()

	t.Run("NoConditions_ReturnsTrue", func(t *testing.T) {
		ok, err := EvaluateLogicConditions(nil, nil)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("WithConditions_NilUserAttrs_ReturnsFalse", func(t *testing.T) {
		cond := buildSimpleLogicCondition(attrID, `"gold"`, enums.LogicalOperatorEQ)
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateLogicConditions_WithUserAttrs - non-nil userAttrs provides actual values
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_WithUserAttrs(t *testing.T) {
	attrID := uuid.New()

	t.Run("Pass_UserAttrMatchesExpected", func(t *testing.T) {
		cond := buildSimpleLogicCondition(attrID, `"gold"`, enums.LogicalOperatorEQ)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"gold"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Fail_UserAttrDoesNotMatchExpected", func(t *testing.T) {
		cond := buildSimpleLogicCondition(attrID, `"gold"`, enums.LogicalOperatorEQ)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"silver"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("MissingAttr_TreatedAsNonMatch", func(t *testing.T) {
		cond := buildSimpleLogicCondition(attrID, `"gold"`, enums.LogicalOperatorEQ)
		// userAttrs provided but does not contain attrID
		userAttrs := map[string]json.RawMessage{
			uuid.New().String(): mustJSON(`"gold"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateLogicConditions_UnifiedPath — regression tests
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_UnifiedPath(t *testing.T) {
	attrID := uuid.New()

	makeLogicCond := func(attrIDStr, expectedVal, logicalOp, dataType string) dto.LogicCondition {
		return dto.LogicCondition{
			ConditionID:     uuid.New().String(),
			AttributeID:     attrIDStr,
			DataType:        dataType,
			LogicalOperator: logicalOp,
			// ConnectorOperator omitted: single-condition callers are last (and only) siblings.
			Sequence:      1,
			ExpectedValue: mustJSON(expectedVal),
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
		cond1.ConnectorOperator = string(enums.ConnectorOperatorAND) // forward-link to cond2

		cond2 := makeLogicCond(attr2.String(), `42`, string(enums.LogicalOperatorGTE), string(enums.AttributeDataTypeNumber))
		cond2.Sequence = 2
		// cond2 is the last sibling: ConnectorOperator must be omitted.

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
		cond1.ConnectorOperator = string(enums.ConnectorOperatorAND) // forward-link to cond2
		cond2 := makeLogicCond(attr2.String(), `100`, string(enums.LogicalOperatorGTE), string(enums.AttributeDataTypeNumber))
		cond2.Sequence = 2
		// cond2 is the last sibling: ConnectorOperator must be omitted.

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
// TestEvaluateLogicConditions_InvalidTree_ReturnsFalse
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_InvalidTree_ReturnsFalse(t *testing.T) {
	attrID := uuid.New()

	t.Run("LastSiblingForwardLink_ReturnsFalse", func(t *testing.T) {
		// c2 is the last sibling and must omit ConnectorOperator.
		c1 := dto.LogicCondition{
			ConditionID:       uuid.New().String(),
			AttributeID:       uuidPtr(attrID).String(),
			DataType:          string(enums.AttributeDataTypeText),
			LogicalOperator:   string(enums.LogicalOperatorEQ),
			ConnectorOperator: string(enums.ConnectorOperatorAND),
			Sequence:          1,
			ExpectedValue:     mustJSON(`"gold"`),
		}
		c2 := dto.LogicCondition{
			ConditionID:       uuid.New().String(),
			AttributeID:       uuidPtr(attrID).String(),
			DataType:          string(enums.AttributeDataTypeText),
			LogicalOperator:   string(enums.LogicalOperatorEQ),
			ConnectorOperator: string(enums.ConnectorOperatorOR),
			Sequence:          2,
			ExpectedValue:     mustJSON(`"gold"`),
		}
		userAttrs := map[string]json.RawMessage{attrID.String(): mustJSON(`"gold"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{c1, c2}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok, "invalid tree must return false, not evaluate")
	})
}

// c1.ConnectorOperator = AND means "combine c1's result with c2 using AND".
// The last sibling omits ConnectorOperator entirely.

func TestEvaluateConditionGroup_MixedConnectorPrecedence(t *testing.T) {
	textAttr := &entity.Attribute{DataType: enums.AttributeDataTypeText}

	makeCondition := func(attrID uuid.UUID, sequence int, connector enums.ConnectorOperator) entity.RuleCondition {
		c := entity.RuleCondition{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     uuidPtr(attrID),
			LogicalOperator: enums.LogicalOperatorEQ,
			Attribute:       textAttr,
			Sequence:        sequence,
		}
		if connector != "" {
			c.ConnectorOperator = connectorPtr(connector)
		}
		return c
	}

	evaluate := func(t *testing.T, values []bool, connectors []enums.ConnectorOperator) bool {
		t.Helper()
		require.Len(t, values, len(connectors)+1)

		conditions := make([]entity.RuleCondition, 0, len(values))
		expected := make(map[string]json.RawMessage, len(values))
		user := make(map[string]json.RawMessage, len(values))

		for i, pass := range values {
			attrID := uuid.New()
			connector := enums.ConnectorOperator("")
			if i < len(connectors) {
				connector = connectors[i]
			}
			conditions = append(conditions, makeCondition(attrID, i+1, connector))
			expected[attrID.String()] = mustJSON(`"yes"`)
			if pass {
				user[attrID.String()] = mustJSON(`"yes"`)
			} else {
				user[attrID.String()] = mustJSON(`"no"`)
			}
		}

		result, err := evaluateConditionGroup(
			conditions,
			NewParsedExpectedValues(expected),
			NewParsedUserAttrs(user),
		)
		require.NoError(t, err)
		return result
	}

	t.Run("AndPairsOrTogether", func(t *testing.T) {
		tests := []struct {
			name       string
			a, b, c, d bool
			want       bool
		}{
			{"all_true", true, true, true, true, true},
			{"left_pair_true", true, true, false, true, true},
			{"right_pair_true", false, true, true, true, true},
			{"all_false", false, false, false, false, false},
			{"left_T_F_or_F_T", true, false, false, true, false},
			{"left_T_F_or_T_T", true, false, true, true, true},
			{"left_pair_true_right_run_false", true, true, true, false, true},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got := evaluate(
					t,
					[]bool{tc.a, tc.b, tc.c, tc.d},
					[]enums.ConnectorOperator{
						enums.ConnectorOperatorAND,
						enums.ConnectorOperatorOR,
						enums.ConnectorOperatorAND,
					},
				)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("OrBeforeAnd", func(t *testing.T) {
		got := evaluate(
			t,
			[]bool{true, false, false},
			[]enums.ConnectorOperator{
				enums.ConnectorOperatorOR,
				enums.ConnectorOperatorAND,
			},
		)
		assert.True(t, got, "T OR F AND F must evaluate as T OR (F AND F)")
	})
}

func TestEvaluateLogicConditions_MixedConnectorUserScenario(t *testing.T) {
	attrA := uuid.New()
	attrB := uuid.New()
	attrC := uuid.New()
	attrD := uuid.New()

	conditions := []dto.LogicCondition{
		{
			ConditionID:       uuid.New().String(),
			AttributeID:       attrA.String(),
			DataType:          string(enums.AttributeDataTypeText),
			LogicalOperator:   string(enums.LogicalOperatorEQ),
			ConnectorOperator: string(enums.ConnectorOperatorAND),
			Sequence:          1,
			ExpectedValue:     mustJSON(`"3"`),
		},
		{
			ConditionID:       uuid.New().String(),
			AttributeID:       attrB.String(),
			DataType:          string(enums.AttributeDataTypeNumber),
			LogicalOperator:   string(enums.LogicalOperatorGTE),
			ConnectorOperator: string(enums.ConnectorOperatorOR),
			Sequence:          2,
			ExpectedValue:     mustJSON(`4000`),
		},
		{
			ConditionID:       uuid.New().String(),
			AttributeID:       attrC.String(),
			DataType:          string(enums.AttributeDataTypeText),
			LogicalOperator:   string(enums.LogicalOperatorEQ),
			ConnectorOperator: string(enums.ConnectorOperatorAND),
			Sequence:          3,
			ExpectedValue:     mustJSON(`"500"`),
		},
		{
			ConditionID:     uuid.New().String(),
			AttributeID:     attrD.String(),
			DataType:        string(enums.AttributeDataTypeText),
			LogicalOperator: string(enums.LogicalOperatorIN),
			Sequence:        4,
			ExpectedValue:   mustJSON(`["70"]`),
		},
	}
	userAttrs := map[string]json.RawMessage{
		attrA.String(): mustJSON(`"3"`),
		attrB.String(): mustJSON(`5000`),
		attrC.String(): mustJSON(`"not-500"`),
		attrD.String(): mustJSON(`"not-70"`),
	}

	ok, err := EvaluateLogicConditions(conditions, userAttrs)
	require.NoError(t, err)
	assert.True(t, ok, `A = "3" AND B >= 4000 should match regardless of the right AND-pair`)
}

func TestEvaluateConditionGroup_NestedPrecedence(t *testing.T) {
	tier := uuid.New()
	age := uuid.New()
	score := uuid.New()
	tierAttr := &entity.Attribute{DataType: enums.AttributeDataTypeText}
	ageAttr := &entity.Attribute{DataType: enums.AttributeDataTypeNumber}
	scoreAttr := &entity.Attribute{DataType: enums.AttributeDataTypeNumber}

	t.Run("ForwardLink_AND_TwoPass", func(t *testing.T) {
		c1 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		c2 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, AttributeID: uuidPtr(age), Sequence: 2,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: ageAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`30`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`35`),
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{c1, c2}, expected, user)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("ForwardLink_AND_FirstFails", func(t *testing.T) {
		c1 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		c2 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, AttributeID: uuidPtr(age), Sequence: 2,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: ageAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`30`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String(): mustJSON(`"silver"`), // fails
			age.String():  mustJSON(`35`),
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{c1, c2}, expected, user)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("ForwardLink_OR_SecondPasses", func(t *testing.T) {
		c1 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorOR),
		}
		c2 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, AttributeID: uuidPtr(age), Sequence: 2,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: ageAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`30`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String(): mustJSON(`"silver"`), // fails
			age.String():  mustJSON(`35`),       // passes
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{c1, c2}, expected, user)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("OwnCheckPlusChildren_ChildConnector_AND_BothPass", func(t *testing.T) {
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: parentID}, AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		child := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, ParentRuleConditionID: &parentID,
			AttributeID: uuidPtr(age), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: ageAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`30`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`35`),
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{parent, child}, expected, user)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("OwnCheckPlusChildren_ChildConnector_AND_OwnFails", func(t *testing.T) {
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: parentID}, AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		child := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, ParentRuleConditionID: &parentID,
			AttributeID: uuidPtr(age), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: ageAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`30`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String(): mustJSON(`"silver"`), // fails
			age.String():  mustJSON(`35`),
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{parent, child}, expected, user)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("OwnCheckPlusChildren_ChildConnector_OR_OwnFails_ChildPasses", func(t *testing.T) {
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: parentID}, AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorOR),
		}
		child := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, ParentRuleConditionID: &parentID,
			AttributeID: uuidPtr(age), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: ageAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String(): mustJSON(`"gold"`),
			age.String():  mustJSON(`30`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String(): mustJSON(`"silver"`), // fails
			age.String():  mustJSON(`35`),       // passes
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{parent, child}, expected, user)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("PureGroup_NoOwnCheck_ChildrenDetermineResult", func(t *testing.T) {
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: parentID}, Sequence: 1,
			// No AttributeID = pure container. No ChildConnectorOperator needed.
		}
		child1 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, ParentRuleConditionID: &parentID,
			AttributeID: uuidPtr(tier), Sequence: 1,
			LogicalOperator: enums.LogicalOperatorEQ, Attribute: tierAttr,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		child2 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()}, ParentRuleConditionID: &parentID,
			AttributeID: uuidPtr(score), Sequence: 2,
			LogicalOperator: enums.LogicalOperatorGTE, Attribute: scoreAttr,
		}
		expected := NewParsedExpectedValues(map[string]json.RawMessage{
			tier.String():  mustJSON(`"gold"`),
			score.String(): mustJSON(`80`),
		})
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			tier.String():  mustJSON(`"gold"`),
			score.String(): mustJSON(`90`),
		})
		result, err := evaluateConditionGroup([]entity.RuleCondition{parent, child1, child2}, expected, user)
		require.NoError(t, err)
		assert.True(t, result)
	})
}

// ---------------------------------------------------------------------------
// TestParsedUserAttrs_ConcurrentAccess — go test -race must pass
// ---------------------------------------------------------------------------

func TestParsedUserAttrs_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	// ParsedUserAttrs is documented as single-goroutine per request.
	// This test verifies the removed mutex does not hide a real race:
	// each goroutine must use its own instance, matching request-scoped evaluation.
	const goroutines = 20
	attrs := map[string]json.RawMessage{"tier": mustJSON(`"gold"`)}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			p := NewParsedUserAttrs(attrs) // own instance per goroutine
			v, ok := p.GetString("tier")
			assert.True(t, ok)
			assert.Equal(t, "gold", v)
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// TestParsedUserAttrs — parse-once semantics and cross-type independence
// ---------------------------------------------------------------------------

func TestParsedUserAttrs_ParseOnce(t *testing.T) {
	t.Parallel()

	t.Run("GetString_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`"hello"`)})
		v1, ok1 := p.GetString("k")
		v2, ok2 := p.GetString("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, "hello", v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetNumber_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`42.5`)})
		v1, ok1 := p.GetNumber("k")
		v2, ok2 := p.GetNumber("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, 42.5, v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetBool_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`true`)})
		v1, ok1 := p.GetBool("k")
		v2, ok2 := p.GetBool("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetDate_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`"2026-04-22"`)})
		v1, ok1 := p.GetDate("k")
		v2, ok2 := p.GetDate("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, v1, v2)
	})

	// Retry-prevention: after a failed parse, swapping the raw bytes must NOT cause
	// the second call to succeed — the attempted flag must block the re-parse.
	t.Run("GetString_noRetry_onFailure", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`123`)}) // number ≠ string
		_, ok1 := p.GetString("k")
		p.cache["k"].raw = mustJSON(`"now-valid"`) // mutate underlying raw
		_, ok2 := p.GetString("k")
		assert.False(t, ok1, "first call must fail on invalid input")
		assert.False(t, ok2, "second call must NOT retry — strAttempted blocks re-parse")
	})

	t.Run("GetNumber_noRetry_onFailure", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`"not-a-number"`)})
		_, ok1 := p.GetNumber("k")
		p.cache["k"].raw = mustJSON(`99`)
		_, ok2 := p.GetNumber("k")
		assert.False(t, ok1)
		assert.False(t, ok2, "second call must NOT retry — numAttempted blocks re-parse")
	})

	t.Run("GetBool_noRetry_onFailure", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`"not-a-bool"`)})
		_, ok1 := p.GetBool("k")
		p.cache["k"].raw = mustJSON(`true`)
		_, ok2 := p.GetBool("k")
		assert.False(t, ok1)
		assert.False(t, ok2, "second call must NOT retry — boolAttempted blocks re-parse")
	})

	t.Run("GetDate_noRetry_onFailure", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`"not-a-date"`)})
		_, ok1 := p.GetDate("k")
		p.cache["k"].raw = mustJSON(`"2026-04-22"`)
		_, ok2 := p.GetDate("k")
		assert.False(t, ok1)
		assert.False(t, ok2, "second call must NOT retry — dateAttempted blocks re-parse")
	})

	t.Run("CrossType_independence", func(t *testing.T) {
		// Calling GetString first on a number value must not affect GetNumber.
		p := NewParsedUserAttrs(map[string]json.RawMessage{"k": mustJSON(`42`)})
		_, strOK := p.GetString("k")
		num, numOK := p.GetNumber("k")
		assert.False(t, strOK, "string parse must fail for a bare number")
		assert.True(t, numOK, "number parse must succeed independently of string attempt")
		assert.Equal(t, float64(42), num)
	})

	t.Run("MissingKey_returnsNotOK", func(t *testing.T) {
		p := NewParsedUserAttrs(map[string]json.RawMessage{})
		_, ok := p.GetString("missing")
		assert.False(t, ok)
	})

	t.Run("NilReceiver_allGetters_returnFalse", func(t *testing.T) {
		var p *ParsedUserAttrs
		_, ok1 := p.GetString("k")
		_, ok2 := p.GetNumber("k")
		_, ok3 := p.GetBool("k")
		_, ok4 := p.GetDate("k")
		assert.False(t, ok1)
		assert.False(t, ok2)
		assert.False(t, ok3)
		assert.False(t, ok4)
	})
}

// ---------------------------------------------------------------------------
// TestParsedExpectedValues_ParseOnce — parse-once semantics for expected values
// ---------------------------------------------------------------------------

func TestParsedExpectedValues_ParseOnce(t *testing.T) {
	t.Parallel()

	t.Run("GetString_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`"gold"`)})
		v1, ok1 := p.GetString("k")
		v2, ok2 := p.GetString("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, "gold", v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetStringSlice_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`["a","b","c"]`)})
		v1, ok1 := p.GetStringSlice("k")
		v2, ok2 := p.GetStringSlice("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, []string{"a", "b", "c"}, v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetNumber_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`99.5`)})
		v1, ok1 := p.GetNumber("k")
		v2, ok2 := p.GetNumber("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, 99.5, v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetNumberSlice_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`[1,2,3]`)})
		v1, ok1 := p.GetNumberSlice("k")
		v2, ok2 := p.GetNumberSlice("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, []float64{1, 2, 3}, v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetBool_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`false`)})
		v1, ok1 := p.GetBool("k")
		v2, ok2 := p.GetBool("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.False(t, v1)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetDate_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`"2026-04-22"`)})
		v1, ok1 := p.GetDate("k")
		v2, ok2 := p.GetDate("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, v1, v2)
	})

	t.Run("GetDateBounds_parsedOnce_onSuccess", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`["2026-01-01","2026-12-31"]`)})
		v1, ok1 := p.GetDateBounds("k")
		v2, ok2 := p.GetDateBounds("k")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, v1, v2)
	})

	// Retry-prevention: after a failed parse, swapping the raw bytes must NOT
	// cause the second call to succeed — the attempted flag must block re-parse.
	t.Run("GetString_noRetry_onFailure", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`42`)})
		_, ok1 := p.GetString("k")
		p.cache["k"].raw = mustJSON(`"now-valid"`)
		_, ok2 := p.GetString("k")
		assert.False(t, ok1)
		assert.False(t, ok2, "strAttempted must block re-parse")
	})

	t.Run("GetNumber_noRetry_onFailure", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`"not-a-number"`)})
		_, ok1 := p.GetNumber("k")
		p.cache["k"].raw = mustJSON(`99`)
		_, ok2 := p.GetNumber("k")
		assert.False(t, ok1)
		assert.False(t, ok2, "numAttempted must block re-parse")
	})

	t.Run("CrossOperator_independence_scalarAndSlice", func(t *testing.T) {
		// GetNumber (scalar) and GetNumberSlice (IN/BETWEEN) share the same raw bytes
		// but cache independently — a number scalar raw must not satisfy a slice parse.
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`42`)})
		num, numOK := p.GetNumber("k")
		_, sliceOK := p.GetNumberSlice("k")
		assert.True(t, numOK)
		assert.Equal(t, float64(42), num)
		assert.False(t, sliceOK, "scalar JSON must not parse as []float64")
	})

	t.Run("Has_missingKey_returnsFalse", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{})
		assert.False(t, p.Has("missing"))
	})

	t.Run("Has_presentKey_returnsTrue", func(t *testing.T) {
		p := NewParsedExpectedValues(map[string]json.RawMessage{"k": mustJSON(`"v"`)})
		assert.True(t, p.Has("k"))
	})

	t.Run("NilReceiver_Has_returnsFalse", func(t *testing.T) {
		var p *ParsedExpectedValues
		assert.False(t, p.Has("k"))
		_, ok := p.GetString("k")
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
			ConditionID:     uuid.New().String(),
			AttributeID:     attrID.String(),
			DataType:        string(enums.AttributeDataTypeText),
			LogicalOperator: string(enums.LogicalOperatorEQ),
			Sequence:        1,
			ExpectedValue:   ev,
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

func TestParsedExpectedValues_Raw(t *testing.T) {
	raw := map[string]json.RawMessage{
		"a": json.RawMessage(`"ANY"`),
		"b": json.RawMessage(`42`),
	}
	p := NewParsedExpectedValues(raw)

	t.Run("PresentKey_ReturnsRawAndTrue", func(t *testing.T) {
		v, ok := p.Raw("a")
		assert.True(t, ok)
		assert.Equal(t, json.RawMessage(`"ANY"`), v)
	})

	t.Run("MissingKey_ReturnsNilAndFalse", func(t *testing.T) {
		v, ok := p.Raw("missing")
		assert.False(t, ok)
		assert.Nil(t, v)
	})

	t.Run("NilReceiver_ReturnsNilAndFalse", func(t *testing.T) {
		var p *ParsedExpectedValues
		v, ok := p.Raw("a")
		assert.False(t, ok)
		assert.Nil(t, v)
	})
}

func TestParsedUserAttrs_IsNull(t *testing.T) {
	attrs := map[string]json.RawMessage{
		"present_string": json.RawMessage(`"gold"`),
		"present_null":   json.RawMessage(`null`),
		"present_spaced": json.RawMessage(` null `), // whitespace-padded null
	}
	p := NewParsedUserAttrs(attrs)

	t.Run("AbsentKey_IsNull", func(t *testing.T) {
		assert.True(t, p.IsNull("missing_absent"))
	})
	t.Run("RawJsonNull_IsNull", func(t *testing.T) {
		assert.True(t, p.IsNull("present_null"))
	})
	t.Run("WhitespacePaddedNull_IsNull", func(t *testing.T) {
		assert.True(t, p.IsNull("present_spaced"))
	})
	t.Run("RawString_IsNotNull", func(t *testing.T) {
		assert.False(t, p.IsNull("present_string"))
	})
	t.Run("NilReceiver_IsNull", func(t *testing.T) {
		var p *ParsedUserAttrs
		assert.True(t, p.IsNull("anything"))
	})
}

func TestEvaluateSingleCondition_SentinelIntegration(t *testing.T) {
	attrID := uuid.New()
	buildCond := func(expectedJSON string, op enums.LogicalOperator, dt enums.AttributeDataType) dto.LogicCondition {
		return dto.LogicCondition{
			ConditionID:     uuid.New().String(),
			AttributeID:     attrID.String(),
			DataType:        string(dt),
			LogicalOperator: string(op),
			Sequence:        1,
			ExpectedValue:   mustJSON(expectedJSON),
		}
	}

	t.Run("ANY_Text_AlwaysMatches", func(t *testing.T) {
		cond := buildCond(`"ANY"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)
		ua := map[string]json.RawMessage{attrID.String(): mustJSON(`"anything"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, ua)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("NULL_Text_MatchesWhenUserAbsent", func(t *testing.T) {
		cond := buildCond(`"NULL"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)
		ua := map[string]json.RawMessage{uuid.New().String(): mustJSON(`"x"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, ua)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("NULL_Text_NoMatchWhenUserHasValue", func(t *testing.T) {
		cond := buildCond(`"NULL"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)
		ua := map[string]json.RawMessage{attrID.String(): mustJSON(`"gold"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, ua)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("CaretList_Text_EQ_PromotedToIN", func(t *testing.T) {
		cond := buildCond(`"gold^silver^bronze"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)
		ua := map[string]json.RawMessage{attrID.String(): mustJSON(`"silver"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, ua)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("CaretList_Number_IN", func(t *testing.T) {
		cond := buildCond(`"10^20^30"`, enums.LogicalOperatorIN, enums.AttributeDataTypeNumber)
		ua := map[string]json.RawMessage{attrID.String(): mustJSON(`20`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, ua)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Date_ANY_FallsThroughToNormalComparatorError", func(t *testing.T) {
		cond := buildCond(`"ANY"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeDate)
		ua := map[string]json.RawMessage{attrID.String(): mustJSON(`"2026-01-01"`)}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, ua)
		require.Error(t, err)
		assert.False(t, ok)
	})
}

func TestEvaluateLogicConditions_Sentinels(t *testing.T) {
	attrID := uuid.New().String()
	makeCond := func(ev string, op enums.LogicalOperator, dt enums.AttributeDataType) dto.LogicCondition {
		return dto.LogicCondition{
			ConditionID:     uuid.New().String(),
			AttributeID:     attrID,
			DataType:        string(dt),
			LogicalOperator: string(op),
			Sequence:        1,
			ExpectedValue:   json.RawMessage(ev),
		}
	}

	t.Run("ANY_AlwaysMatches", func(t *testing.T) {
		ok, err := EvaluateLogicConditions(
			[]dto.LogicCondition{makeCond(`"ANY"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)},
			map[string]json.RawMessage{attrID: json.RawMessage(`"whatever"`)},
		)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("NULL_MatchesAbsentUser", func(t *testing.T) {
		ok, err := EvaluateLogicConditions(
			[]dto.LogicCondition{makeCond(`"NULL"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)},
			map[string]json.RawMessage{},
		)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("CaretList_IN_Promoted", func(t *testing.T) {
		ok, err := EvaluateLogicConditions(
			[]dto.LogicCondition{makeCond(`"a^b^c"`, enums.LogicalOperatorEQ, enums.AttributeDataTypeText)},
			map[string]json.RawMessage{attrID: json.RawMessage(`"b"`)},
		)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateLogicConditions_CONTAINS — end-to-end CONTAINS operator
// ---------------------------------------------------------------------------

func TestEvaluateLogicConditions_CONTAINS(t *testing.T) {
	attrID := uuid.New()

	makeCond := func(expected string) dto.LogicCondition {
		return dto.LogicCondition{
			ConditionID:     uuid.New().String(),
			AttributeID:     attrID.String(),
			DataType:        string(enums.AttributeDataTypeText),
			LogicalOperator: string(enums.LogicalOperatorCONTAINS),
			Sequence:        1,
			ExpectedValue:   mustJSON(expected),
		}
	}

	t.Run("Pass_SubstringFound", func(t *testing.T) {
		cond := makeCond(`"gold"`)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"golden card"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Fail_SubstringNotFound", func(t *testing.T) {
		cond := makeCond(`"gold"`)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"silver tier"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Pass_CaseInsensitive", func(t *testing.T) {
		cond := makeCond(`"GOLD"`)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"golden card"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Pass_ExactMatch", func(t *testing.T) {
		cond := makeCond(`"gold"`)
		userAttrs := map[string]json.RawMessage{
			attrID.String(): mustJSON(`"gold"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Fail_MissingUserAttr", func(t *testing.T) {
		cond := makeCond(`"gold"`)
		userAttrs := map[string]json.RawMessage{
			uuid.New().String(): mustJSON(`"gold"`),
		}
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, userAttrs)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Fail_NilUserAttrs", func(t *testing.T) {
		cond := makeCond(`"gold"`)
		ok, err := EvaluateLogicConditions([]dto.LogicCondition{cond}, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
