package evaluator

import (
	"fmt"

	"kbank-ecms/internal/domain/entity"
)

// ValidateConditionTree enforces the invariants required by the new
// childConnectorOperator / forward-link semantics:
//
//  1. A node that has BOTH an own leaf check (AttributeID != nil) AND
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
		if parent.AttributeID != nil && parent.ChildConnectorOperator == nil {
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
