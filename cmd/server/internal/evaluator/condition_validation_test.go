package evaluator

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

func TestValidateConditionTree(t *testing.T) {
	t.Run("EmptyConditions_OK", func(t *testing.T) {
		require.NoError(t, ValidateConditionTree(nil))
	})

	t.Run("SingleRoot_OK", func(t *testing.T) {
		c := entity.RuleCondition{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     uuidPtr(uuid.New()),
			LogicalOperator: enums.LogicalOperatorEQ,
			Sequence:        1,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{c}))
	})

	t.Run("RootSiblings_ForwardLink_OK", func(t *testing.T) {
		// Forward-link: c1.ConnectorOperator -> joins c1 with c2; c2 (last) omits.
		c1 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			AttributeID:       uuidPtr(uuid.New()),
			Sequence:          1,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		c2 := entity.RuleCondition{
			BaseModel:   entity.BaseModel{ID: uuid.New()},
			AttributeID: uuidPtr(uuid.New()),
			Sequence:    2,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{c1, c2}))
	})

	t.Run("RootSiblings_MixedConnector_Error", func(t *testing.T) {
		c1 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			Sequence:          1,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		c2 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			Sequence:          2,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorOR),
		}
		c3 := entity.RuleCondition{
			BaseModel: entity.BaseModel{ID: uuid.New()},
			Sequence:  3,
			// last sibling: no connector
		}
		err := ValidateConditionTree([]entity.RuleCondition{c1, c2, c3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mixed")
	})

	t.Run("RootSibling_MissingForwardLink_Error", func(t *testing.T) {
		// Two roots: c1 must carry ConnectorOperator (forward-link). c2 (last) must not.
		c1 := entity.RuleCondition{BaseModel: entity.BaseModel{ID: uuid.New()}, Sequence: 1} // missing forward-link
		c2 := entity.RuleCondition{BaseModel: entity.BaseModel{ID: uuid.New()}, Sequence: 2}
		err := ValidateConditionTree([]entity.RuleCondition{c1, c2})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ConnectorOperator")
	})

	t.Run("LastSibling_HasForwardLink_Error", func(t *testing.T) {
		c1 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			Sequence:          1,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		c2 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			Sequence:          2,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND), // last sibling MUST omit
		}
		err := ValidateConditionTree([]entity.RuleCondition{c1, c2})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "last sibling")
	})

	t.Run("Group_OwnCheckPlusChildren_MissingChildConnector_Error", func(t *testing.T) {
		// Parent has its own leaf check (AttributeID set) AND children → ChildConnectorOperator required.
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel:       entity.BaseModel{ID: parentID},
			AttributeID:     uuidPtr(uuid.New()),
			LogicalOperator: enums.LogicalOperatorEQ,
			Sequence:        1,
			// ChildConnectorOperator missing
		}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuidPtr(uuid.New()),
			Sequence:              1,
		}
		err := ValidateConditionTree([]entity.RuleCondition{parent, child1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ChildConnectorOperator")
	})

	t.Run("PureGroup_NoOwnCheck_NoChildConnectorRequired_OK", func(t *testing.T) {
		// AttributeID = nil = pure group container; ChildConnectorOperator not required.
		parentID := uuid.New()
		parent := entity.RuleCondition{BaseModel: entity.BaseModel{ID: parentID}, Sequence: 1}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuidPtr(uuid.New()),
			Sequence:              1,
			ConnectorOperator:     connectorPtr(enums.ConnectorOperatorOR),
		}
		child2 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuidPtr(uuid.New()),
			Sequence:              2,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{parent, child1, child2}))
	})

	t.Run("Group_OwnCheckPlusChildren_WithChildConnector_OK", func(t *testing.T) {
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel:              entity.BaseModel{ID: parentID},
			AttributeID:            uuidPtr(uuid.New()),
			LogicalOperator:        enums.LogicalOperatorEQ,
			Sequence:               1,
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuidPtr(uuid.New()),
			Sequence:              1,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{parent, child1}))
	})

	t.Run("ErrorIncludesBuildLogicExpressionWithExpectedValues", func(t *testing.T) {
		attrA := uuid.New()
		attrA1 := uuid.New()
		attrA2 := uuid.New()
		attrB := uuid.New()
		parentID := uuid.New()

		parent := entity.RuleCondition{
			BaseModel:              entity.BaseModel{ID: parentID},
			AttributeID:            &attrA,
			Attribute:              &entity.Attribute{DisplayName: "A"},
			LogicalOperator:        enums.LogicalOperatorEQ,
			Sequence:               1,
			ConnectorOperator:      connectorPtr(enums.ConnectorOperatorAND),
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           &attrA1,
			Attribute:             &entity.Attribute{DisplayName: "A1"},
			LogicalOperator:       enums.LogicalOperatorGT,
			Sequence:              1,
			ConnectorOperator:     connectorPtr(enums.ConnectorOperatorOR),
		}
		child2 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           &attrA2,
			Attribute:             &entity.Attribute{DisplayName: "A2"},
			LogicalOperator:       enums.LogicalOperatorLT,
			Sequence:              2,
		}
		lastWithDanglingConnector := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			AttributeID:       &attrB,
			Attribute:         &entity.Attribute{DisplayName: "B"},
			LogicalOperator:   enums.LogicalOperatorEQ,
			Sequence:          2,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorOR),
		}

		expectedValues := map[string]json.RawMessage{
			attrA.String():  json.RawMessage(`"v1"`),
			attrA1.String(): json.RawMessage(`"v2"`),
			attrA2.String(): json.RawMessage(`"v3"`),
			attrB.String():  json.RawMessage(`"v4"`),
		}

		err := ValidateConditionTreeWithExpectedValues(
			[]entity.RuleCondition{lastWithDanglingConnector, child2, parent, child1},
			expectedValues,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "last sibling")
		assert.Contains(t, err.Error(), "condition: "+BuildLogicExpression(
			[]entity.RuleCondition{lastWithDanglingConnector, child2, parent, child1},
			expectedValues,
		))
	})

	t.Run("ErrorUsesBuildLogicExpressionForChildConnectorValidation", func(t *testing.T) {
		attrParent := uuid.New()
		attrChild := uuid.New()
		parentID := uuid.New()

		parent := entity.RuleCondition{
			BaseModel:       entity.BaseModel{ID: parentID},
			AttributeID:     &attrParent,
			Attribute:       &entity.Attribute{FieldName: "parent_field"},
			LogicalOperator: enums.LogicalOperatorEQ,
			Sequence:        1,
		}
		child := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           &attrChild,
			Attribute:             &entity.Attribute{DisplayName: "Child"},
			LogicalOperator:       enums.LogicalOperatorIN,
			Sequence:              1,
		}
		expectedValues := map[string]json.RawMessage{
			attrParent.String(): json.RawMessage(`["x", "y"]`),
			attrChild.String():  json.RawMessage(`["z"]`),
		}
		conditions := []entity.RuleCondition{parent, child}

		err := ValidateConditionTreeWithExpectedValues(conditions, expectedValues)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ChildConnectorOperator")
		assert.Contains(t, err.Error(), "condition: "+BuildLogicExpression(conditions, expectedValues))
	})
}
