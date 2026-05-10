package evaluator

import (
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
			AttributeID:     uuid.New(),
			LogicalOperator: enums.LogicalOperatorEQ,
			Sequence:        1,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{c}))
	})

	t.Run("RootSiblings_ForwardLink_OK", func(t *testing.T) {
		// Forward-link: c1.ConnectorOperator -> joins c1 with c2; c2 (last) omits.
		c1 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			AttributeID:       uuid.New(),
			Sequence:          1,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		c2 := entity.RuleCondition{
			BaseModel:   entity.BaseModel{ID: uuid.New()},
			AttributeID: uuid.New(),
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
			AttributeID:     uuid.New(),
			LogicalOperator: enums.LogicalOperatorEQ,
			Sequence:        1,
			// ChildConnectorOperator missing
		}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuid.New(),
			Sequence:              1,
		}
		err := ValidateConditionTree([]entity.RuleCondition{parent, child1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ChildConnectorOperator")
	})

	t.Run("PureGroup_NoOwnCheck_NoChildConnectorRequired_OK", func(t *testing.T) {
		// AttributeID = uuid.Nil = pure group container; ChildConnectorOperator not required.
		parentID := uuid.New()
		parent := entity.RuleCondition{BaseModel: entity.BaseModel{ID: parentID}, Sequence: 1}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuid.New(),
			Sequence:              1,
			ConnectorOperator:     connectorPtr(enums.ConnectorOperatorOR),
		}
		child2 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuid.New(),
			Sequence:              2,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{parent, child1, child2}))
	})

	t.Run("Group_OwnCheckPlusChildren_WithChildConnector_OK", func(t *testing.T) {
		parentID := uuid.New()
		parent := entity.RuleCondition{
			BaseModel:              entity.BaseModel{ID: parentID},
			AttributeID:            uuid.New(),
			LogicalOperator:        enums.LogicalOperatorEQ,
			Sequence:               1,
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           uuid.New(),
			Sequence:              1,
		}
		require.NoError(t, ValidateConditionTree([]entity.RuleCondition{parent, child1}))
	})
}
