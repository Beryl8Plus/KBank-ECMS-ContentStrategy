package evaluator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// GenerateConditionHash generates a stable SHA-256 hash for a set of rule conditions.
// It normalizes the condition tree to ensure the same logic results in the same hash.
func GenerateConditionHash(conditions []entity.RuleCondition) (string, error) {
	if len(conditions) == 0 {
		return "", nil
	}

	// Build parent -> children index. Key "" represents root (no parent).
	byParent := make(map[string][]entity.RuleCondition, len(conditions))
	for _, c := range conditions {
		key := ""
		if c.ParentRuleConditionID != nil {
			key = c.ParentRuleConditionID.String()
		}
		byParent[key] = append(byParent[key], c)
	}

	// Sort each sibling group by Sequence, then by ID for deterministic order.
	for k := range byParent {
		sort.Slice(byParent[k], func(i, j int) bool {
			if byParent[k][i].Sequence != byParent[k][j].Sequence {
				return byParent[k][i].Sequence < byParent[k][j].Sequence
			}
			return byParent[k][i].ID.String() < byParent[k][j].ID.String()
		})
	}

	roots := byParent[""]
	if len(roots) == 0 {
		return "", nil
	}

	// Example canonical string for a condition: "attribute_id:operator:compact_json_value"
	// Single leaf:
	// attribute-1:EQ:42

	// Two root siblings (default connector -> AND):
	// attribute-1:EQ:"foo" AND attribute-2:NEQ:{"min":10}

	// Nested group with connectors:
	// (attribute-3:GT:5 AND attribute-4:LT:10) OR attribute-5:IN:["a","b"]
	canonicalStr := buildCanonicalString(byParent, roots)

	hash := sha256.Sum256([]byte(canonicalStr))
	return hex.EncodeToString(hash[:]), nil
}

// buildCanonicalString recursively builds a deterministic string representation of the condition tree.
// Forward-link semantics: siblings[i].ConnectorOperator joins siblings[i] with siblings[i+1].
func buildCanonicalString(byParent map[string][]entity.RuleCondition, siblings []entity.RuleCondition) string {
	var parts []string
	for i, c := range siblings {
		nodeStr := renderCanonicalNode(byParent, c)
		parts = append(parts, nodeStr)
		// Forward-link: siblings[i].ConnectorOperator joins siblings[i] with siblings[i+1].
		if i < len(siblings)-1 {
			parts = append(parts, forwardConnector(c.ConnectorOperator))
		}
	}
	return strings.Join(parts, " ")
}

// renderCanonicalNode renders a single node, handling three cases:
//  1. Pure leaf (own check, no children): "attr_uuid:operator"
//  2. Pure group container (no own check, has children): "(children)"
//  3. Mixed node (own check + children): "attr_uuid:operator [ChildConnectorOp] (children)"
func renderCanonicalNode(byParent map[string][]entity.RuleCondition, c entity.RuleCondition) string {
	children := byParent[c.ID.String()]
	hasOwnCheck := c.AttributeID != uuid.Nil

	if len(children) == 0 {
		if !hasOwnCheck {
			return "()" // pure group with no children — edge case
		}
		return fmt.Sprintf("%s:%s", c.AttributeID.String(), c.LogicalOperator)
	}

	childrenStr := "(" + buildCanonicalString(byParent, children) + ")"

	if !hasOwnCheck {
		// Pure group container: children determine the full result.
		return childrenStr
	}

	// Mixed node: own check combined with children via ChildConnectorOperator.
	ownStr := fmt.Sprintf("%s:%s", c.AttributeID.String(), c.LogicalOperator)
	return ownStr + " " + childConnectorOf(c.ChildConnectorOperator) + " " + childrenStr
}

// forwardConnector returns the connector string for the forward-link between siblings.
func forwardConnector(op *enums.ConnectorOperator) string {
	if op == nil {
		return "AND"
	}
	return string(*op)
}

// childConnectorOf returns the string representation of a node's ChildConnectorOperator.
func childConnectorOf(op *enums.ConnectorOperator) string {
	if op == nil {
		return "AND"
	}
	return string(*op)
}

// BuildLogicExpression builds a value-aware canonical string for a set of rule
// conditions together with their expected values (attr UUID → compact JSON value).
// This is used to generate a stable hash that identifies both the logic structure
// AND the expected attribute values, enabling per-user evaluation caching.
//
// Leaf format: attr_uuid:operator:compact_json_value
// Example:     attr_uuid_1:=:"A"   or   attr_uuid_2:>:10   or   attr_uuid_3:IN:["v1","v2"]
//
// Returns an empty string when conditions is empty.
func BuildLogicExpression(conditions []entity.RuleCondition, expectedValues map[string]json.RawMessage) string {
	if len(conditions) == 0 {
		return ""
	}

	// Build parent -> children index. Key "" represents root (no parent).
	byParent := make(map[string][]entity.RuleCondition, len(conditions))
	for _, c := range conditions {
		key := ""
		if c.ParentRuleConditionID != nil {
			key = c.ParentRuleConditionID.String()
		}
		byParent[key] = append(byParent[key], c)
	}

	// Sort each sibling group by Sequence, then by ID for deterministic order.
	for k := range byParent {
		sort.Slice(byParent[k], func(i, j int) bool {
			if byParent[k][i].Sequence != byParent[k][j].Sequence {
				return byParent[k][i].Sequence < byParent[k][j].Sequence
			}
			return byParent[k][i].ID.String() < byParent[k][j].ID.String()
		})
	}

	roots := byParent[""]
	if len(roots) == 0 {
		return ""
	}

	return buildValueCanonicalString(byParent, roots, expectedValues)
}

// GenerateLogicHash returns the SHA-256 hex hash of BuildLogicExpression output.
// Returns ("", nil) when conditions is empty.
func GenerateLogicHash(conditions []entity.RuleCondition, expectedValues map[string]json.RawMessage) (string, error) {
	expr := BuildLogicExpression(conditions, expectedValues)
	if expr == "" {
		return "", nil
	}
	sum := sha256.Sum256([]byte(expr))
	return hex.EncodeToString(sum[:]), nil
}

// buildValueCanonicalString recursively builds a canonical string with expected values
// embedded in each leaf node.
// Forward-link semantics: siblings[i].ConnectorOperator joins siblings[i] with siblings[i+1].
func buildValueCanonicalString(byParent map[string][]entity.RuleCondition, siblings []entity.RuleCondition, expectedValues map[string]json.RawMessage) string {
	var parts []string
	for i, c := range siblings {
		nodeStr := renderValueCanonicalNode(byParent, c, expectedValues)
		parts = append(parts, nodeStr)
		// Forward-link: siblings[i].ConnectorOperator joins siblings[i] with siblings[i+1].
		if i < len(siblings)-1 {
			parts = append(parts, forwardConnector(c.ConnectorOperator))
		}
	}
	return strings.Join(parts, " ")
}

// renderValueCanonicalNode renders a single node with embedded expected values,
// handling three cases:
//  1. Pure leaf (own check, no children): "attr_uuid:operator:compact_json_value"
//  2. Pure group container (no own check, has children): "(children)"
//  3. Mixed node (own check + children): "attr_uuid:operator:value [ChildConnectorOp] (children)"
func renderValueCanonicalNode(byParent map[string][]entity.RuleCondition, c entity.RuleCondition, expectedValues map[string]json.RawMessage) string {
	children := byParent[c.ID.String()]
	hasOwnCheck := c.AttributeID != uuid.Nil

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

	if len(children) == 0 {
		if !hasOwnCheck {
			return "()" // pure group with no children — edge case
		}
		return leafStr()
	}

	childrenStr := "(" + buildValueCanonicalString(byParent, children, expectedValues) + ")"

	if !hasOwnCheck {
		return childrenStr
	}

	// Mixed node: own check combined with children via ChildConnectorOperator.
	return leafStr() + " " + childConnectorOf(c.ChildConnectorOperator) + " " + childrenStr
}
