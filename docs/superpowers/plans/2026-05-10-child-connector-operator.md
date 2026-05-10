# Rule Condition `ChildConnectorOperator` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the evaluator/normalizer match the new node model so `(A = v1 AND (A1 = v2 OR A2 >= v3)) AND B < v4` evaluates with correct precedence.

**Node model (per the canonical JSON example):**

```json
{
  "conditions": [
    {
      "attributeId": "A", "logicalOperator": "=", "value": "v1",
      "childConnectorOperator": "AND",          // joins (A=v1) with the children-group below
      "connectorOperator": "AND",               // forward-link to next root sibling (B)
      "conditions": [
        { "attributeId": "A1", "logicalOperator": "=",  "value": "v2", "connectorOperator": "OR" }, // forward-link A1 -> A2
        { "attributeId": "A2", "logicalOperator": ">=", "value": "v3" }                              // last child, no connectorOperator
      ]
    },
    { "attributeId": "B", "logicalOperator": "<", "value": "v4" } // last root sibling, no connectorOperator
  ]
}
```

**Field semantics (final, replacing my earlier assumptions):**
- A node may simultaneously carry an own leaf check (`attributeId`, `logicalOperator`, `value`) **and** children.
- `childConnectorOperator` on a node = operator between **this node's own leaf check** and **the combined result of its children**. Required iff the node has both an own check AND children. (When AttributeID is `uuid.Nil` the node is a pure group: skip the own check and return children-combined directly. When the node has no children: `childConnectorOperator` is unused.)
- `connectorOperator` on a node = **forward-link** to the next sibling in the same parent. The last sibling in a chain MUST omit it; every other sibling MUST set it.
- All forward-links inside one sibling chain must share the same value — no mixing at one level. To mix AND/OR, nest a group.

**Evaluation order (for the canonical example):**
1. Inner children of node A: `A1 OR A2` (using A1.connectorOperator).
2. Node A: `(A=v1) AND (A1 OR A2)` (using A.childConnectorOperator).
3. Root chain: `nodeA AND B<v4` (using A.connectorOperator).

**Tech Stack:** Go 1.x, GORM (Postgres), testify, gofakeit, go-sqlmock. No new deps.

---

## File Structure

**Modify:**
- `internal/domain/entity/rule_condition.go` — already has the field; add `IsGroup()` helper if not present (skip if covered by AttributeID nil convention).
- `cmd/server/internal/evaluator/condition_evaluator.go` — `evalSiblings`, `logicConditionToRuleCondition`.
- `cmd/server/internal/evaluator/condition_normalization.go` — `buildCanonicalString`, `buildValueCanonicalString`.
- `cmd/server/internal/evaluator/placement_logic.go` — `buildLogicEntry`: also stamp `ChildConnectorOperator` on each `LogicCondition`.
- `internal/delivery/http/dto/content.go` — `LogicCondition` gets `ChildConnectorOperator string` JSON field.
- `scripts/mockdata/generate_decision_rule_mock.go` — `mockSet` and `ruleConditionValues` add `CHILD_CONNECTOR_OPERATOR` column.
- `cmd/server/internal/evaluator/evaluator_test.go`, `condition_evaluator_test.go`, `condition_normalization_test.go` — update existing tests for pointer `ConnectorOperator` and add new coverage.

**Create:**
- A validation helper in `cmd/server/internal/evaluator/condition_validation.go` to enforce the invariants below at evaluation time (returns error so the caller can decide).
- `cmd/server/internal/evaluator/condition_validation_test.go`.

**Invariants enforced:**
1. A node that has BOTH an own leaf check (AttributeID != uuid.Nil) AND children MUST set `ChildConnectorOperator`.
2. In any sibling chain (root or inner), every non-last sibling MUST set `ConnectorOperator` (forward-link). The last sibling MUST omit it.
3. All forward-link `ConnectorOperator` values within a single sibling chain MUST be equal (no mixing at one level — nest a group instead).

---

## Task 1: Add `ChildConnectorOperator` to `LogicCondition` DTO

**Files:**
- Modify: `internal/delivery/http/dto/content.go:143-152`

- [ ] **Step 1: Edit `LogicCondition` to add the new field**

Replace the struct (preserve existing fields):

```go
type LogicCondition struct {
	ConditionID            string          `json:"conditionId"`
	ParentConditionID      string          `json:"parentConditionId,omitempty"` // empty = root
	AttributeID            string          `json:"attributeId"`
	DataType               string          `json:"dataType"`
	LogicalOperator        string          `json:"logicalOperator"`
	ConnectorOperator      string          `json:"connectorOperator,omitempty"`      // root-level only
	ChildConnectorOperator string          `json:"childConnectorOperator,omitempty"` // group-level: connector for all direct children
	Sequence               int             `json:"sequence"`
	ExpectedValue          json.RawMessage `json:"expectedValue"`
}
```

- [ ] **Step 2: Build to confirm compile**

Run: `go build ./...`
Expected: PASS (no other code reads new field yet).

- [ ] **Step 3: Commit**

```bash
git add internal/delivery/http/dto/content.go
git commit -m "feat(content-dto): add ChildConnectorOperator to LogicCondition"
```

---

## Task 2: Compile-fix all callers for pointer `ConnectorOperator`

The entity already changed `ConnectorOperator` from value to pointer. This task fixes every existing caller so the project compiles before logic changes.

**Files:**
- Modify: `cmd/server/internal/evaluator/condition_evaluator.go:388, 605-619`
- Modify: `cmd/server/internal/evaluator/condition_normalization.go:78, 171`
- Modify: `cmd/server/internal/evaluator/placement_logic.go:56-73`
- Modify: `cmd/server/internal/evaluator/evaluator_test.go:56`
- Modify: `cmd/server/internal/evaluator/condition_evaluator_test.go:38`
- Modify: `cmd/server/internal/evaluator/condition_normalization_test.go:31, 38, 64, 80, 120-128, 142-143`

- [ ] **Step 1: Add a pointer helper in evaluator package**

Add to `cmd/server/internal/evaluator/condition_evaluator.go` near the bottom of the file (above `parseDate`):

```go
// connectorPtr returns &op (helper for building *enums.ConnectorOperator literals).
func connectorPtr(op enums.ConnectorOperator) *enums.ConnectorOperator {
	return &op
}

// connectorValue safely dereferences a *enums.ConnectorOperator, returning
// the zero ConnectorOperator ("") when p is nil.
func connectorValue(p *enums.ConnectorOperator) enums.ConnectorOperator {
	if p == nil {
		return ""
	}
	return *p
}
```

- [ ] **Step 2: Update `evalSiblings` connector read**

In `cmd/server/internal/evaluator/condition_evaluator.go:382-394`, replace the existing inner block:

```go
	for i := 1; i < len(siblings); i++ {
		c := siblings[i]
		val, err := evalNode(byParent, c, depth, expectedVals, parsed)
		if err != nil {
			return false, err
		}
		if connectorValue(c.ConnectorOperator) == enums.ConnectorOperatorOR {
			result = result || val
		} else {
			result = result && val
		}
	}
```

(Logic unchanged for now — Task 4 rewrites this to honor `ChildConnectorOperator`.)

- [ ] **Step 3: Update `logicConditionToRuleCondition`**

In `cmd/server/internal/evaluator/condition_evaluator.go:605-619`, replace the body:

```go
func logicConditionToRuleCondition(lc dto.LogicCondition) entity.RuleCondition {
	id, _ := uuid.Parse(lc.ConditionID)
	rc := entity.RuleCondition{
		BaseModel:       entity.BaseModel{ID: id},
		AttributeID:     mustParseUUID(lc.AttributeID),
		Sequence:        lc.Sequence,
		LogicalOperator: enums.LogicalOperator(lc.LogicalOperator),
		Attribute:       &entity.Attribute{DataType: enums.AttributeDataType(lc.DataType)},
	}
	if lc.ConnectorOperator != "" {
		op := enums.ConnectorOperator(lc.ConnectorOperator)
		rc.ConnectorOperator = &op
	}
	if lc.ChildConnectorOperator != "" {
		op := enums.ConnectorOperator(lc.ChildConnectorOperator)
		rc.ChildConnectorOperator = &op
	}
	if lc.ParentConditionID != "" {
		pid, _ := uuid.Parse(lc.ParentConditionID)
		rc.ParentRuleConditionID = &pid
	}
	return rc
}
```

- [ ] **Step 4: Update normalization helpers**

In `cmd/server/internal/evaluator/condition_normalization.go:75-86`, replace:

```go
		if i > 0 {
			connector := string(connectorValue(c.ConnectorOperator))
			if connector == "" {
				connector = "AND"
			}
			parts = append(parts, connector)
		}
```

In `cmd/server/internal/evaluator/condition_normalization.go:168-178`, replace identically:

```go
		if i > 0 {
			connector := string(connectorValue(c.ConnectorOperator))
			if connector == "" {
				connector = "AND"
			}
			parts = append(parts, connector)
		}
```

(Logic still pulls from sibling here; Task 5 rewrites these to honor parent's `ChildConnectorOperator`.)

- [ ] **Step 5: Update `placement_logic.go` LogicCondition stamping**

In `cmd/server/internal/evaluator/placement_logic.go:56-73`, replace the loop body:

```go
	conditions := make([]dto.LogicCondition, 0, len(rule.RuleConditions))
	for _, rc := range rule.RuleConditions {
		lc := dto.LogicCondition{
			ConditionID:            rc.ID.String(),
			AttributeID:            rc.AttributeID.String(),
			LogicalOperator:        string(rc.LogicalOperator),
			ConnectorOperator:      string(connectorValue(rc.ConnectorOperator)),
			ChildConnectorOperator: string(connectorValue(rc.ChildConnectorOperator)),
			Sequence:               rc.Sequence,
			ExpectedValue:          expectedValues[rc.AttributeID.String()],
		}
		if rc.ParentRuleConditionID != nil {
			lc.ParentConditionID = rc.ParentRuleConditionID.String()
		}
		if rc.Attribute != nil {
			lc.DataType = string(rc.Attribute.DataType)
		}
		conditions = append(conditions, lc)
	}
```

- [ ] **Step 6: Update test struct literals to use `connectorPtr`**

`cmd/server/internal/evaluator/evaluator_test.go:56` change:

```go
		ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
```

`cmd/server/internal/evaluator/condition_evaluator_test.go:38` change:

```go
		ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
```

`cmd/server/internal/evaluator/condition_normalization_test.go:31, 38, 64, 80` — change each `ConnectorOperator: enums.ConnectorOperatorAND` (or OR) to `ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND)` (preserving AND vs OR per existing line).

`cmd/server/internal/evaluator/condition_normalization_test.go:120-128`, replace the helper:

```go
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
```

- [ ] **Step 7: Run full build + tests**

Run: `go build ./... && go test ./...`
Expected: PASS — same logic, only types adjusted.

- [ ] **Step 8: Commit**

```bash
git add cmd/server/internal/evaluator internal/delivery/http/dto
git commit -m "refactor(rule-condition): adapt callers to pointer ConnectorOperator"
```

---

## Task 3: Add validation helper for the new invariants

**Files:**
- Create: `cmd/server/internal/evaluator/condition_validation.go`
- Test:   `cmd/server/internal/evaluator/condition_validation_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/server/internal/evaluator/condition_validation_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server/internal/evaluator/ -run TestValidateConditionTree -v`
Expected: FAIL — `ValidateConditionTree` undefined.

- [ ] **Step 3: Implement `ValidateConditionTree`**

Create `cmd/server/internal/evaluator/condition_validation.go`:

```go
package evaluator

import (
	"fmt"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// ValidateConditionTree enforces the invariants required by the new
// childConnectorOperator / forward-link semantics:
//
//  1. A node that has BOTH an own leaf check (AttributeID != uuid.Nil) AND
//     children MUST set ChildConnectorOperator (it joins own_check with the
//     children-combined result).
//  2. In any sibling chain (root or inner), every non-last sibling MUST set
//     ConnectorOperator (forward-link to the next sibling) and the last
//     sibling MUST omit it.
//  3. All forward-link ConnectorOperator values within one sibling chain must
//     share the same value. Mixing AND/OR at one level is disallowed; nest a
//     group instead.
//
// Returns the first violation encountered.
func ValidateConditionTree(conditions []entity.RuleCondition) error {
	if len(conditions) == 0 {
		return nil
	}

	byParent := make(map[string][]entity.RuleCondition, len(conditions))
	parentByID := make(map[string]entity.RuleCondition, len(conditions))
	for _, c := range conditions {
		key := ""
		if c.ParentRuleConditionID != nil {
			key = c.ParentRuleConditionID.String()
		}
		byParent[key] = append(byParent[key], c)
		parentByID[c.ID.String()] = c
	}

	// Rule 1: own-check + children → ChildConnectorOperator required.
	for parentKey, children := range byParent {
		if parentKey == "" || len(children) == 0 {
			continue
		}
		parent, ok := parentByID[parentKey]
		if !ok {
			continue // dangling parent reference; not this validator's concern
		}
		if parent.AttributeID != uuid.Nil && parent.ChildConnectorOperator == nil {
			return fmt.Errorf("rule_condition %s has own check and children but ChildConnectorOperator is unset", parent.ID)
		}
	}

	// Rules 2 + 3: forward-link uniformity in every sibling chain.
	for _, siblings := range byParent {
		if err := validateSiblingChain(siblings); err != nil {
			return err
		}
	}

	return nil
}

func validateSiblingChain(siblings []entity.RuleCondition) error {
	if len(siblings) < 2 {
		// Single sibling: connector must be unset (it has no next sibling).
		if len(siblings) == 1 && siblings[0].ConnectorOperator != nil {
			return fmt.Errorf("rule_condition %s is the last sibling and must omit ConnectorOperator", siblings[0].ID)
		}
		return nil
	}

	sorted := make([]entity.RuleCondition, len(siblings))
	copy(sorted, siblings)
	sortConditionsBySequence(sorted)

	var ref *string
	for i := 0; i < len(sorted); i++ {
		isLast := i == len(sorted)-1
		c := sorted[i]
		if isLast {
			if c.ConnectorOperator != nil {
				return fmt.Errorf("rule_condition %s is the last sibling and must omit ConnectorOperator", c.ID)
			}
			continue
		}
		if c.ConnectorOperator == nil {
			return fmt.Errorf("rule_condition %s must set ConnectorOperator (forward-link to next sibling)", c.ID)
		}
		val := string(*c.ConnectorOperator)
		if ref == nil {
			ref = &val
			continue
		}
		if *ref != val {
			return fmt.Errorf("rule_condition %s has mixed sibling ConnectorOperator (%s vs %s); nest a group instead", c.ID, *ref, val)
		}
	}
	return nil
}

func sortConditionsBySequence(cs []entity.RuleCondition) {
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && cs[j-1].Sequence > cs[j].Sequence; j-- {
			cs[j-1], cs[j] = cs[j], cs[j-1]
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/server/internal/evaluator/ -run TestValidateConditionTree -v`
Expected: PASS for all sub-tests.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/internal/evaluator/condition_validation.go cmd/server/internal/evaluator/condition_validation_test.go
git commit -m "feat(evaluator): validate group ChildConnectorOperator and uniform root connector"
```

---

## Task 4: Rewrite `evalNode` / `evalSiblings` for forward-link + own-check-plus-children

The current `evalSiblings` reads each non-first sibling's `ConnectorOperator` as a backward-link and folds; `evalNode` only evaluates a leaf OR descends into a pure group, never both. Both behaviors are wrong under the new model:

- A node can be **both** a leaf and a parent. `evalNode(c)` must compute `own_check (c.ChildConnectorOperator) children_combined` when both are present.
- `ConnectorOperator` is a **forward-link**. In a chain `[c0, c1, c2]`, the connector that joins `c0` to `c1` lives on `c0` (not `c1`); the last sibling carries no connector. Validated as uniform, so picking `siblings[0].ConnectorOperator` is unambiguous.
- A pure group node (AttributeID == uuid.Nil) returns `children_combined` directly; the own check is skipped.

**Files:**
- Modify: `cmd/server/internal/evaluator/condition_evaluator.go:356-404`
- Test:   `cmd/server/internal/evaluator/condition_evaluator_test.go` (add new tests)

- [ ] **Step 1: Write the failing tests**

Append to `cmd/server/internal/evaluator/condition_evaluator_test.go`:

```go
// TestEvaluateConditionGroup_NestedPrecedence verifies the canonical example:
//   (A = v1 AND (A1 = v2 OR A2 >= v3)) AND B < v4
// where node A carries BOTH its own leaf check (A = v1) AND children (A1, A2)
// joined by A.ChildConnectorOperator = AND. Inner siblings A1->A2 are linked
// forward by A1.ConnectorOperator = OR. Root siblings A->B are linked forward
// by A.ConnectorOperator = AND.
func TestEvaluateConditionGroup_NestedPrecedence(t *testing.T) {
	attrA := uuid.New()
	attrA1 := uuid.New()
	attrA2 := uuid.New()
	attrB := uuid.New()
	nodeAID := uuid.New()

	conds := []entity.RuleCondition{
		// Root sibling 0: node A — leaf check A=v1 AND its children-group.
		{
			BaseModel:              entity.BaseModel{ID: nodeAID},
			AttributeID:            attrA,
			LogicalOperator:        enums.LogicalOperatorEQ,
			Sequence:               1,
			ConnectorOperator:      connectorPtr(enums.ConnectorOperatorAND), // forward-link to B
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND), // joins (A=v1) with (A1 OR A2)
			Attribute:              &entity.Attribute{DataType: enums.AttributeDataTypeText},
		},
		// Inner sibling 0: A1.
		{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &nodeAID,
			AttributeID:           attrA1,
			LogicalOperator:       enums.LogicalOperatorEQ,
			Sequence:              1,
			ConnectorOperator:     connectorPtr(enums.ConnectorOperatorOR), // forward-link A1 -> A2
			Attribute:             &entity.Attribute{DataType: enums.AttributeDataTypeText},
		},
		// Inner sibling 1 (last): A2 — no ConnectorOperator.
		{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &nodeAID,
			AttributeID:           attrA2,
			LogicalOperator:       enums.LogicalOperatorGTE,
			Sequence:              2,
			Attribute:             &entity.Attribute{DataType: enums.AttributeDataTypeNumber},
		},
		// Root sibling 1 (last): B — no ConnectorOperator.
		{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     attrB,
			LogicalOperator: enums.LogicalOperatorLT,
			Sequence:        2,
			Attribute:       &entity.Attribute{DataType: enums.AttributeDataTypeNumber},
		},
	}

	expected := NewParsedExpectedValues(map[string]json.RawMessage{
		attrA.String():  mustJSON(`"v1"`),
		attrA1.String(): mustJSON(`"v2"`),
		attrA2.String(): mustJSON(`10`),
		attrB.String():  mustJSON(`100`),
	})

	t.Run("AllTrue_ViaInnerOR_OuterAND_RootAND", func(t *testing.T) {
		// A=v1 (true), A1=v2 (true), A2>=10 (false; user=5), B<100 (true; user=50)
		// node A = (true) AND (true OR false) = true
		// root  = (true) AND (true)            = true
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			attrA.String():  mustJSON(`"v1"`),
			attrA1.String(): mustJSON(`"v2"`),
			attrA2.String(): mustJSON(`5`),
			attrB.String():  mustJSON(`50`),
		})
		ok, err := evaluateConditionGroup(conds, expected, user)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("InnerOrBothFalse_NodeAFalse_RootFalse", func(t *testing.T) {
		// A=v1 (true), A1!=v2 (false), A2<10 (false; user=5), B<100 (true)
		// node A = (true) AND (false OR false) = false
		// root  = (false) AND (true)           = false
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			attrA.String():  mustJSON(`"v1"`),
			attrA1.String(): mustJSON(`"different"`),
			attrA2.String(): mustJSON(`5`),
			attrB.String():  mustJSON(`50`),
		})
		ok, err := evaluateConditionGroup(conds, expected, user)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("OwnCheckFalse_NodeAFalse_EvenIfChildrenTrue", func(t *testing.T) {
		// A=v1 (false; user="other"), A1=v2 (true), A2>=10 (true), B<100 (true)
		// node A = (false) AND (true OR true) = false
		// root  = (false) AND (true)          = false
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			attrA.String():  mustJSON(`"other"`),
			attrA1.String(): mustJSON(`"v2"`),
			attrA2.String(): mustJSON(`50`),
			attrB.String():  mustJSON(`50`),
		})
		ok, err := evaluateConditionGroup(conds, expected, user)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("BFails_RootAndFails", func(t *testing.T) {
		// node A true via A2 path; B<100 fails (B=200)
		// root = (true) AND (false) = false
		user := NewParsedUserAttrs(map[string]json.RawMessage{
			attrA.String():  mustJSON(`"v1"`),
			attrA1.String(): mustJSON(`"different"`),
			attrA2.String(): mustJSON(`50`),
			attrB.String():  mustJSON(`200`),
		})
		ok, err := evaluateConditionGroup(conds, expected, user)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server/internal/evaluator/ -run TestEvaluateConditionGroup_NestedPrecedence -v`
Expected: FAILs because (a) the current `evalNode` does not combine own_check with children when both are present, and (b) the current `evalSiblings` reads connectors from the wrong sibling.

- [ ] **Step 3: Rewrite `evaluateConditionGroup`, `evalSiblings`, `evalNode`**

In `cmd/server/internal/evaluator/condition_evaluator.go:356-404`, replace `evaluateConditionGroup`, `evalSiblings`, and `evalNode`:

```go
func evaluateConditionGroup(conditions []entity.RuleCondition, expectedVals *ParsedExpectedValues, parsed *ParsedUserAttrs) (bool, error) {
	byParent := make(map[string][]entity.RuleCondition, len(conditions))
	for _, c := range conditions {
		key := ""
		if c.ParentRuleConditionID != nil {
			key = c.ParentRuleConditionID.String()
		}
		byParent[key] = append(byParent[key], c)
	}
	for k := range byParent {
		sort.Slice(byParent[k], func(i, j int) bool {
			return byParent[k][i].Sequence < byParent[k][j].Sequence
		})
	}
	roots := byParent[""]
	if len(roots) == 0 {
		return true, nil
	}
	return evalSiblings(byParent, roots, 1, expectedVals, parsed)
}

// evalSiblings combines `siblings` left-to-right using a single forward-link
// connector. The connector lives on each non-last sibling's ConnectorOperator;
// ValidateConditionTree guarantees uniformity, so siblings[0].ConnectorOperator
// is canonical. Falls back to AND when unset (defensive).
func evalSiblings(
	byParent map[string][]entity.RuleCondition,
	siblings []entity.RuleCondition,
	depth int,
	expectedVals *ParsedExpectedValues,
	parsed *ParsedUserAttrs,
) (bool, error) {
	result, err := evalNode(byParent, siblings[0], depth, expectedVals, parsed)
	if err != nil {
		return false, err
	}
	if len(siblings) == 1 {
		return result, nil
	}

	connector := enums.ConnectorOperatorAND
	if siblings[0].ConnectorOperator != nil {
		connector = *siblings[0].ConnectorOperator
	}

	for i := 1; i < len(siblings); i++ {
		val, err := evalNode(byParent, siblings[i], depth, expectedVals, parsed)
		if err != nil {
			return false, err
		}
		if connector == enums.ConnectorOperatorOR {
			result = result || val
		} else {
			result = result && val
		}
	}
	return result, nil
}

// evalNode evaluates a single condition node, accounting for the new model:
//   - own check: present iff AttributeID != uuid.Nil. If present, evaluate it.
//   - children: present iff byParent[c.ID] is non-empty. If present (and depth
//     budget allows), evaluate them combined via forward-link siblings.
//   - When both are present, combine via c.ChildConnectorOperator
//     (defaults to AND when unset).
//   - When neither is present, return false (degenerate row).
func evalNode(
	byParent map[string][]entity.RuleCondition,
	c entity.RuleCondition,
	depth int,
	expectedVals *ParsedExpectedValues,
	parsed *ParsedUserAttrs,
) (bool, error) {
	hasOwn := c.AttributeID != uuid.Nil
	var children []entity.RuleCondition
	if depth < maxConditionDepth {
		children = byParent[c.ID.String()]
	}

	switch {
	case hasOwn && len(children) > 0:
		ownResult, err := evaluateSingleCondition(c, expectedVals, parsed)
		if err != nil {
			return false, err
		}
		childResult, err := evalSiblings(byParent, children, depth+1, expectedVals, parsed)
		if err != nil {
			return false, err
		}
		connector := enums.ConnectorOperatorAND
		if c.ChildConnectorOperator != nil {
			connector = *c.ChildConnectorOperator
		}
		if connector == enums.ConnectorOperatorOR {
			return ownResult || childResult, nil
		}
		return ownResult && childResult, nil

	case hasOwn:
		return evaluateSingleCondition(c, expectedVals, parsed)

	case len(children) > 0:
		return evalSiblings(byParent, children, depth+1, expectedVals, parsed)

	default:
		// Pure group node with no children at this depth (depth budget exhausted
		// or empty subtree) and no own check — treat as non-match.
		return false, nil
	}
}
```

- [ ] **Step 4: Run all evaluator tests**

Run: `go test ./cmd/server/internal/evaluator/ -v`
Expected: PASS — including the new `TestEvaluateConditionGroup_NestedPrecedence` sub-tests AND the existing legacy tests (which use only root-level siblings, so `siblings[1].ConnectorOperator` still drives them).

- [ ] **Step 5: Commit**

```bash
git add cmd/server/internal/evaluator/condition_evaluator.go cmd/server/internal/evaluator/condition_evaluator_test.go
git commit -m "fix(evaluator): use parent ChildConnectorOperator for inner-group siblings"
```

---

## Task 5: Update normalization (hash + canonical expression)

The two canonical builders must mirror the new evaluation order:
- For nodes with own_check + children: render `(own_leaf <CHILD_CONNECTOR> (children_combined))`.
- Inside any sibling chain, connectors are forward-link: read from `siblings[i].ConnectorOperator` and emitted between `siblings[i]` and `siblings[i+1]`. The last sibling has no connector. Validation guarantees uniformity, so emitted connectors are consistent within a chain.

This change WILL change hash output for any pre-existing rule that has nested siblings — intentional, because the structure now evaluates differently.

**Files:**
- Modify: `cmd/server/internal/evaluator/condition_normalization.go:62-88, 145-181`
- Test:   `cmd/server/internal/evaluator/condition_normalization_test.go` (add new test)

- [ ] **Step 1: Write the failing test**

Append to `cmd/server/internal/evaluator/condition_normalization_test.go`:

```go
func TestBuildCanonicalString_OwnCheckPlusChildren(t *testing.T) {
	attrA := uuid.New()
	attrA1 := uuid.New()
	attrA2 := uuid.New()
	attrB := uuid.New()
	nodeAID := uuid.New()

	conds := []entity.RuleCondition{
		{
			BaseModel:              entity.BaseModel{ID: nodeAID},
			AttributeID:            attrA,
			LogicalOperator:        enums.LogicalOperatorEQ,
			Sequence:               1,
			ConnectorOperator:      connectorPtr(enums.ConnectorOperatorAND), // forward-link to B
			ChildConnectorOperator: connectorPtr(enums.ConnectorOperatorAND), // joins A's own check with its children
		},
		{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &nodeAID,
			AttributeID:           attrA1,
			LogicalOperator:       enums.LogicalOperatorEQ,
			Sequence:              1,
			ConnectorOperator:     connectorPtr(enums.ConnectorOperatorOR), // forward-link A1 -> A2
		},
		{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			ParentRuleConditionID: &nodeAID,
			AttributeID:           attrA2,
			LogicalOperator:       enums.LogicalOperatorGTE,
			Sequence:              2,
		},
		{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     attrB,
			LogicalOperator: enums.LogicalOperatorLT,
			Sequence:        2,
		},
	}

	hash, err := GenerateConditionHash(conds)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Flipping nodeA's ChildConnectorOperator must change the hash.
	condsAlt := append([]entity.RuleCondition(nil), conds...)
	condsAlt[0].ChildConnectorOperator = connectorPtr(enums.ConnectorOperatorOR)
	hashAlt, _ := GenerateConditionHash(condsAlt)
	assert.NotEqual(t, hash, hashAlt, "ChildConnectorOperator must affect hash")

	// Flipping the inner forward-link (A1.ConnectorOperator) must change the hash.
	condsForward := append([]entity.RuleCondition(nil), conds...)
	condsForward[1].ConnectorOperator = connectorPtr(enums.ConnectorOperatorAND)
	hashForward, _ := GenerateConditionHash(condsForward)
	assert.NotEqual(t, hash, hashForward, "inner forward-link connector must affect hash")

	// Setting last sibling's ConnectorOperator (which is invalid) MUST NOT change the hash —
	// the normalizer reads forward-link from non-last siblings only.
	condsTail := append([]entity.RuleCondition(nil), conds...)
	condsTail[2].ConnectorOperator = connectorPtr(enums.ConnectorOperatorOR)
	hashTail, _ := GenerateConditionHash(condsTail)
	assert.Equal(t, hash, hashTail, "last sibling's ConnectorOperator must be ignored")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server/internal/evaluator/ -run TestBuildCanonicalString_OwnCheckPlusChildren -v`
Expected: FAIL — current builder ignores `ChildConnectorOperator` and reads connector from the right sibling (backward-link).

- [ ] **Step 3: Rewrite both canonical builders**

In `cmd/server/internal/evaluator/condition_normalization.go`, replace the entire `buildCanonicalString` function (around lines 62-88):

```go
func buildCanonicalString(byParent map[string][]entity.RuleCondition, siblings []entity.RuleCondition) string {
	var parts []string
	for i, c := range siblings {
		parts = append(parts, renderCanonicalNode(byParent, c))
		if i < len(siblings)-1 {
			parts = append(parts, forwardConnector(c))
		}
	}
	return strings.Join(parts, " ")
}

// renderCanonicalNode produces the canonical fragment for a single node:
//   - leaf only:                       "attrID:OP"
//   - children only (pure group):      "(child1 CONN child2 ...)"
//   - own check + children:            "(attrID:OP <CHILD_CONN> (child1 CONN child2 ...))"
func renderCanonicalNode(byParent map[string][]entity.RuleCondition, c entity.RuleCondition) string {
	hasOwn := c.AttributeID != uuid.Nil
	children := byParent[c.ID.String()]

	switch {
	case hasOwn && len(children) > 0:
		ownStr := fmt.Sprintf("%s:%s", c.AttributeID.String(), c.LogicalOperator)
		childStr := "(" + buildCanonicalString(byParent, children) + ")"
		return "(" + ownStr + " " + childConnectorOf(c) + " " + childStr + ")"
	case hasOwn:
		return fmt.Sprintf("%s:%s", c.AttributeID.String(), c.LogicalOperator)
	case len(children) > 0:
		return "(" + buildCanonicalString(byParent, children) + ")"
	default:
		return "()"
	}
}

// forwardConnector returns the connector that links node c to its NEXT sibling.
// Falls back to "AND" when unset (defensive; ValidateConditionTree should reject this).
func forwardConnector(c entity.RuleCondition) string {
	if c.ConnectorOperator != nil {
		return string(*c.ConnectorOperator)
	}
	return "AND"
}

// childConnectorOf returns the connector joining a node's own check with its
// children-combined result. Falls back to "AND" when unset.
func childConnectorOf(c entity.RuleCondition) string {
	if c.ChildConnectorOperator != nil {
		return string(*c.ChildConnectorOperator)
	}
	return "AND"
}
```

Replace `buildValueCanonicalString` (around lines 145-181):

```go
func buildValueCanonicalString(byParent map[string][]entity.RuleCondition, siblings []entity.RuleCondition, expectedValues map[string]json.RawMessage) string {
	var parts []string
	for i, c := range siblings {
		parts = append(parts, renderValueCanonicalNode(byParent, c, expectedValues))
		if i < len(siblings)-1 {
			parts = append(parts, forwardConnector(c))
		}
	}
	return strings.Join(parts, " ")
}

func renderValueCanonicalNode(byParent map[string][]entity.RuleCondition, c entity.RuleCondition, expectedValues map[string]json.RawMessage) string {
	hasOwn := c.AttributeID != uuid.Nil
	children := byParent[c.ID.String()]

	leafStr := func() string {
		valStr := "null"
		if raw, ok := expectedValues[c.AttributeID.String()]; ok {
			var v interface{}
			if err := json.Unmarshal(raw, &v); err == nil {
				if compacted, err := json.Marshal(v); err == nil {
					valStr = string(compacted)
				}
			}
		}
		return fmt.Sprintf("%s:%s:%s", c.AttributeID.String(), c.LogicalOperator, valStr)
	}

	switch {
	case hasOwn && len(children) > 0:
		childStr := "(" + buildValueCanonicalString(byParent, children, expectedValues) + ")"
		return "(" + leafStr() + " " + childConnectorOf(c) + " " + childStr + ")"
	case hasOwn:
		return leafStr()
	case len(children) > 0:
		return "(" + buildValueCanonicalString(byParent, children, expectedValues) + ")"
	default:
		return "()"
	}
}
```

You will also need to add `"github.com/google/uuid"` to the import block in `condition_normalization.go` (it currently lacks it).

- [ ] **Step 4: Update legacy normalization tests**

The existing `TestGenerateConditionHash/NestedGroups` test in `condition_normalization_test.go` uses backward-link semantics (e.g., `condParent.ConnectorOperator: enums.ConnectorOperatorOR` to mean "OR before condParent"). Under the new semantics that connector is a forward-link from `condParent` to its next sibling — but `condParent` is the LAST root in that test. Update the test so:
- `cond1.ConnectorOperator = connectorPtr(enums.ConnectorOperatorOR)` (forward-link cond1 -> condParent)
- `condParent.ConnectorOperator = nil` (it is the last root sibling)
- Inside the inner group: `child1.ConnectorOperator = connectorPtr(enums.ConnectorOperatorAND)` (forward-link child1 -> child2), `child2.ConnectorOperator = nil`
- Add `condParent.ChildConnectorOperator = connectorPtr(enums.ConnectorOperatorAND)` only if condParent now also has a non-zero AttributeID (it doesn't — keep nil; condParent is a pure group, so ChildConnectorOperator is not required by validation).

Apply the same forward-link rewrite to `TestBuildLogicExpression` cases that pass non-empty connectors (they currently mark the last sibling — flip them to the first sibling).

- [ ] **Step 5: Run normalization tests**

Run: `go test ./cmd/server/internal/evaluator/ -run "TestGenerateConditionHash|TestBuildLogicExpression|TestBuildCanonicalString_OwnCheckPlusChildren" -v`
Expected: PASS — new test plus the rewritten legacy tests.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/internal/evaluator/condition_normalization.go cmd/server/internal/evaluator/condition_normalization_test.go
git commit -m "fix(evaluator): hash group siblings using parent ChildConnectorOperator"
```

---

## Task 6: Wire validation into evaluation entry points

Run `ValidateConditionTree` from `EvaluateRuleScore` and `EvaluateLogicConditions`. On error, log and return non-match (no panic). This catches misconfigured rules without breaking the request pipeline.

**Files:**
- Modify: `cmd/server/internal/evaluator/condition_evaluator.go:291-348`
- Test:   `cmd/server/internal/evaluator/condition_evaluator_test.go` (add new test)

- [ ] **Step 1: Write the failing test**

Append to `cmd/server/internal/evaluator/condition_evaluator_test.go`:

```go
func TestEvaluateLogicConditions_InvalidTree_ReturnsFalse(t *testing.T) {
	// A group with children but no ChildConnectorOperator: must be rejected as non-match.
	parentID := uuid.New()
	parentLC := dto.LogicCondition{
		ConditionID: parentID.String(),
		Sequence:    1,
		// ChildConnectorOperator intentionally empty.
	}
	childLC1 := dto.LogicCondition{
		ConditionID:       uuid.New().String(),
		ParentConditionID: parentID.String(),
		AttributeID:       uuid.New().String(),
		DataType:          string(enums.AttributeDataTypeText),
		LogicalOperator:   string(enums.LogicalOperatorEQ),
		Sequence:          1,
		ExpectedValue:     mustJSON(`"x"`),
	}
	childLC2 := childLC1
	childLC2.ConditionID = uuid.New().String()
	childLC2.AttributeID = uuid.New().String()
	childLC2.Sequence = 2

	ok, err := EvaluateLogicConditions([]dto.LogicCondition{parentLC, childLC1, childLC2}, map[string]json.RawMessage{
		childLC1.AttributeID: mustJSON(`"x"`),
		childLC2.AttributeID: mustJSON(`"x"`),
	})
	require.NoError(t, err)
	assert.False(t, ok, "invalid tree must produce non-match, not an error")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server/internal/evaluator/ -run TestEvaluateLogicConditions_InvalidTree_ReturnsFalse -v`
Expected: FAIL — currently the evaluator silently treats missing ChildConnectorOperator as AND and returns `true`.

- [ ] **Step 3: Hook validation into both entry points**

In `cmd/server/internal/evaluator/condition_evaluator.go`, modify `EvaluateRuleScore` (around line 291): after `parsed := NewParsedUserAttrs(userAttrs)`, add:

```go
	if err := ValidateConditionTree(rule.RuleConditions); err != nil {
		return nil, rule.Score, nil
	}
```

In the same file, modify `EvaluateLogicConditions` (around line 325): after the `rcs := make(...)` loop and before the `evaluateConditionGroup` call, add:

```go
	if err := ValidateConditionTree(rcs); err != nil {
		return false, nil
	}
```

- [ ] **Step 4: Update legacy tests to match forward-link semantics**

Validation now rejects trees that previously passed. Make these specific edits so legacy tests remain valid:

`cmd/server/internal/evaluator/condition_evaluator_test.go`:

- `buildSimpleRule` (line 29) — single-condition rule, the lone condition must NOT carry ConnectorOperator. Change `ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND)` to remove the line entirely (default nil).
- `makeLogicCond` helper (around line 138-148) — strip `ConnectorOperator: string(enums.ConnectorOperatorAND)` from the default; tests that need a forward-link will set it explicitly on the non-last sibling.
- `TwoConditions_AND_BothPass` and `TwoConditions_AND_OneFails` (around lines 180-214) — assign `cond1.ConnectorOperator = string(enums.ConnectorOperatorAND)` (forward-link cond1 -> cond2). Remove `cond2.ConnectorOperator = ...` (cond2 is last).
- `TestEvaluateLogicConditions_NilOrNullExpectedValue/makeCondWithExpected` (around line 490) — drop the default `ConnectorOperator` assignment; the test uses a single condition.

`cmd/server/internal/evaluator/condition_normalization_test.go`:

- `TestGenerateConditionHash/DeterministicSorting` (around line 24-48): `cond1.ConnectorOperator = connectorPtr(enums.ConnectorOperatorAND)` (forward-link cond1 -> cond2); `cond2.ConnectorOperator = nil` (last sibling).
- `TestGenerateConditionHash/NestedGroups` (around line 50-93): there are three roots (cond1, condParent) and inner children (child1, child2). Set:
  - `cond1.ConnectorOperator = connectorPtr(enums.ConnectorOperatorOR)` (forward-link cond1 -> condParent)
  - `condParent.ConnectorOperator = nil` (last root sibling)
  - `child1.ConnectorOperator = connectorPtr(enums.ConnectorOperatorAND)` (forward-link child1 -> child2)
  - `child2.ConnectorOperator = nil` (last child)
  - `condParent.ChildConnectorOperator` stays nil because condParent is a pure group (AttributeID == uuid.Nil).
- `TestBuildLogicExpression/Deterministic` (around line 142-152): mirror — cond1 is non-last, cond2 is last; only cond1 carries ConnectorOperator.

`cmd/server/internal/evaluator/evaluator_test.go:56` — if `buildSimpleRule` or its analogue here uses `ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND)` on a single-condition rule, remove it (single sibling must not carry forward-link).

- [ ] **Step 5: Run all evaluator + handler tests**

Run: `go test ./...`
Expected: PASS — including the new invalid-tree test (returns false, no error) and all updated legacy tests.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/internal/evaluator
git commit -m "feat(evaluator): reject invalid condition trees and migrate legacy tests"
```

---

## Task 7: Update mock data generator

The dev-seed SQL must include the new `CHILD_CONNECTOR_OPERATOR` column so reseeded environments don't break GORM scans on absent columns.

**Files:**
- Modify: `scripts/mockdata/generate_decision_rule_mock.go:95-112, 320-340, 453-456, 555-569`
- Verify: `scripts/mockdata/generate_decision_rule_mock_test.go:128`

- [ ] **Step 1: Add `ChildConnectorOperator` to `mockSet`**

In `scripts/mockdata/generate_decision_rule_mock.go:95-112`, add the field after `ConnectorOperator`:

```go
	ConnectorOperator      string
	ChildConnectorOperator string
```

- [ ] **Step 2: Default-populate it in `buildSets`**

In `scripts/mockdata/generate_decision_rule_mock.go` around line 332 (where `ConnectorOperator: "AND"` is set), add:

```go
				ConnectorOperator:      "AND",
				ChildConnectorOperator: "",
```

(Empty string is valid: every mock condition is a single root leaf with no children, so no group invariant applies.)

- [ ] **Step 3: Add column to INSERT and value row**

In `scripts/mockdata/generate_decision_rule_mock.go:453-456`, replace the `rule_conditions` insert columns:

```go
	writeInsert(&buf, "rule_conditions",
		[]string{`"ID"`, `"SEQUENCE"`, `"DECISION_RULE_ID"`, `"ATTRIBUTE_ID"`, `"LOGICAL_OPERATOR"`, `"CONNECTOR_OPERATOR"`, `"CHILD_CONNECTOR_OPERATOR"`, `"CREATED_AT"`, `"UPDATED_AT"`},
		ruleConditionValues(sets),
	)
```

In `scripts/mockdata/generate_decision_rule_mock.go:555-569`, replace `ruleConditionValues`:

```go
func ruleConditionValues(sets []mockSet) [][]string {
	out := make([][]string, 0, len(sets))
	for _, s := range sets {
		childConnector := "NULL"
		if s.ChildConnectorOperator != "" {
			childConnector = sqlStringLiteral(s.ChildConnectorOperator)
		}
		out = append(out, []string{
			sqlStringLiteral(s.ConditionID),
			"1",
			sqlStringLiteral(s.DecisionRuleID),
			sqlStringLiteral(s.AttributeID),
			sqlStringLiteral(s.LogicalOperator),
			sqlStringLiteral(s.ConnectorOperator),
			childConnector,
			"NOW()", "NOW()",
		})
	}
	return out
}
```

- [ ] **Step 4: Run mock generator tests**

Run: `go test ./scripts/mockdata/ -v`
Expected: PASS — the existing assertion that `INSERT INTO rule_conditions` appears in the output still holds; column-count expectations may need a tweak.

If the test asserts a specific column count or fixed text, adjust the assertion to include `CHILD_CONNECTOR_OPERATOR` (read the test first to know what to change).

- [ ] **Step 5: Commit**

```bash
git add scripts/mockdata/
git commit -m "chore(mockdata): include CHILD_CONNECTOR_OPERATOR column in seed SQL"
```

---

## Task 8: End-to-end verification

- [ ] **Step 1: Format and lint**

Run: `make fmt && make lint`
Expected: clean.

- [ ] **Step 2: Full test suite with race detector**

Run: `go test -race ./...`
Expected: PASS.

- [ ] **Step 3: Build binary (also regenerates Swagger)**

Run: `make build`
Expected: PASS.

- [ ] **Step 4: Smoke-run the service against dev infra**

Run (in two terminals):
```bash
make dev-up
make run
```
Then `curl 'http://localhost:8082/healthz'` → expect 200.
Stop with `make dev-down` afterwards.

- [ ] **Step 5: Commit any formatting changes**

```bash
git add -A
git commit -m "chore: format after ChildConnectorOperator migration" || true
```

---

## Notes for the implementer

- **Why pointer types**: making `ConnectorOperator` and `ChildConnectorOperator` pointers lets us distinguish "explicitly set to AND" from "unset" — necessary so validation can flag missing values rather than silently defaulting.
- **Backward compatibility**: existing data has `ConnectorOperator` populated on inner-group siblings but `ChildConnectorOperator` unset on the parent. After this change, those rules will be flagged invalid by `ValidateConditionTree` and return non-match. A separate, **out-of-scope** data migration must populate `ChildConnectorOperator` on group nodes from the most common child connector before this evaluator change is rolled out — flag this to the team in the PR description.
- **Scope guardrails**: do not touch CLEN clients, cache layers, or DI wiring. The change is confined to entity → evaluator → DTO → mock generator.
