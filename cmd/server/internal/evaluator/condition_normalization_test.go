package evaluator

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

func TestGenerateConditionHash(t *testing.T) {
	attr1 := uuid.New()
	attr2 := uuid.New()

	t.Run("EmptyConditions", func(t *testing.T) {
		hash, err := GenerateConditionHash([]entity.RuleCondition{})
		assert.NoError(t, err)
		assert.Equal(t, "", hash)
	})

	t.Run("DeterministicSorting", func(t *testing.T) {
		// Logic: (Attr1 = ...) AND (Attr2 > ...)
		cond1 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			AttributeID:       attr1,
			LogicalOperator:   enums.LogicalOperatorEQ,
			Sequence:          1,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}
		cond2 := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: uuid.New()},
			AttributeID:       attr2,
			LogicalOperator:   enums.LogicalOperatorGT,
			Sequence:          2,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
		}

		// Hash with order 1, 2
		hash1, _ := GenerateConditionHash([]entity.RuleCondition{cond1, cond2})
		// Hash with order 2, 1
		hash2, _ := GenerateConditionHash([]entity.RuleCondition{cond2, cond1})

		assert.NotEmpty(t, hash1)
		assert.Equal(t, hash1, hash2, "Hash should be deterministic regardless of input slice order")
	})

	t.Run("NestedGroups", func(t *testing.T) {
		parentID := uuid.New()

		// Logic: (Attr1 =) OR (Nested: Attr2 > AND Attr1 !=)
		cond1 := entity.RuleCondition{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     attr1,
			LogicalOperator: enums.LogicalOperatorEQ,
			Sequence:        1,
		}
		// Parent node for nesting
		condParent := entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: parentID},
			Sequence:          2,
			ConnectorOperator: connectorPtr(enums.ConnectorOperatorOR),
		}
		// Children of parentID
		child1 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           attr2,
			LogicalOperator:       enums.LogicalOperatorGT,
			Sequence:              1,
		}
		child2 := entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &parentID,
			AttributeID:           attr1,
			LogicalOperator:       enums.LogicalOperatorNEQ,
			Sequence:              2,
			ConnectorOperator:     connectorPtr(enums.ConnectorOperatorAND),
		}

		hash, err := GenerateConditionHash([]entity.RuleCondition{cond1, condParent, child1, child2})
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)

		// Change operator in nested child — expect different hash
		child2Mod := child2
		child2Mod.LogicalOperator = enums.LogicalOperatorGTE
		hashMod, _ := GenerateConditionHash([]entity.RuleCondition{cond1, condParent, child1, child2Mod})

		assert.NotEqual(t, hash, hashMod, "Hash should change when nested condition operator changes")
	})

	t.Run("DifferentAttributeChangeHash", func(t *testing.T) {
		attr3 := uuid.New()

		cond1 := entity.RuleCondition{
			AttributeID:     attr1,
			LogicalOperator: enums.LogicalOperatorIN,
			Sequence:        1,
		}
		cond2 := entity.RuleCondition{
			AttributeID:     attr3,
			LogicalOperator: enums.LogicalOperatorIN,
			Sequence:        1,
		}

		hash1, _ := GenerateConditionHash([]entity.RuleCondition{cond1})
		hash2, _ := GenerateConditionHash([]entity.RuleCondition{cond2})

		assert.NotEqual(t, hash1, hash2, "Different attributes should produce different hashes")
	})
}

func TestBuildLogicExpression(t *testing.T) {
	attr1 := uuid.New()
	attr2 := uuid.New()

	baseCond := func(attrID uuid.UUID, op enums.LogicalOperator, seq int, connector enums.ConnectorOperator) entity.RuleCondition {
		rc := entity.RuleCondition{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     attrID,
			LogicalOperator: op,
			Sequence:        seq,
		}
		if connector != "" {
			c := connector
			rc.ConnectorOperator = &c
		}
		return rc
	}

	t.Run("SingleLeafEQ", func(t *testing.T) {
		cond := baseCond(attr1, enums.LogicalOperatorEQ, 1, "")
		vals := map[string]json.RawMessage{
			attr1.String(): json.RawMessage(`"hello"`),
		}
		expr := BuildLogicExpression([]entity.RuleCondition{cond}, vals)
		assert.Contains(t, expr, attr1.String()+":")
		assert.Contains(t, expr, ":=:")
		assert.Contains(t, expr, `"hello"`)
	})

	t.Run("Deterministic", func(t *testing.T) {
		cond1 := baseCond(attr1, enums.LogicalOperatorEQ, 1, enums.ConnectorOperatorAND)
		cond2 := baseCond(attr2, enums.LogicalOperatorGT, 2, "")
		vals := map[string]json.RawMessage{
			attr1.String(): json.RawMessage(`"A"`),
			attr2.String(): json.RawMessage(`10`),
		}

		expr1 := BuildLogicExpression([]entity.RuleCondition{cond1, cond2}, vals)
		expr2 := BuildLogicExpression([]entity.RuleCondition{cond2, cond1}, vals)
		assert.Equal(t, expr1, expr2, "Output should be order-independent")
	})

	t.Run("ValueChangeChangesHash", func(t *testing.T) {
		cond := baseCond(attr1, enums.LogicalOperatorEQ, 1, "")
		valsA := map[string]json.RawMessage{attr1.String(): json.RawMessage(`"A"`)}
		valsB := map[string]json.RawMessage{attr1.String(): json.RawMessage(`"B"`)}

		hashA, errA := GenerateLogicHash([]entity.RuleCondition{cond}, valsA)
		hashB, errB := GenerateLogicHash([]entity.RuleCondition{cond}, valsB)
		assert.NoError(t, errA)
		assert.NoError(t, errB)
		assert.NotEqual(t, hashA, hashB, "Changing expected value must change the hash")
	})

	t.Run("SameInputSameHash", func(t *testing.T) {
		cond := baseCond(attr1, enums.LogicalOperatorIN, 1, "")
		vals := map[string]json.RawMessage{attr1.String(): json.RawMessage(`["x","y"]`)}

		h1, _ := GenerateLogicHash([]entity.RuleCondition{cond}, vals)
		h2, _ := GenerateLogicHash([]entity.RuleCondition{cond}, vals)
		assert.Equal(t, h1, h2, "Same input must always produce the same hash")
	})

	t.Run("EmptyConditions", func(t *testing.T) {
		expr := BuildLogicExpression([]entity.RuleCondition{}, nil)
		assert.Empty(t, expr)

		hash, err := GenerateLogicHash([]entity.RuleCondition{}, nil)
		assert.NoError(t, err)
		assert.Empty(t, hash)
	})
}
