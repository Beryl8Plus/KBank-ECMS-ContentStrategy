package evaluator

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainservice "kbank-ecms/internal/domain/service"

	"github.com/google/uuid"
)

// EvaluateRuleScore resolves the effective score for a DecisionRule by evaluating
// its RuleConditions against each Rule variation's expected attribute values
// (sourced from RuleAttributes).
//
// userAttrs carries live user attribute values (attributeID → compact JSON value).
// When non-nil, leaf conditions compare against these values.
// When nil, conditions with user-dependent attributes are treated as non-match
// (the caller is expected to defer real evaluation to delivery time).
//
// Algorithm:
//  1. No conditions → return rule.Score unchanged.
//  2. Evaluate each Rule variation in OrderNo order. Build an expected-value map
//     from the variation's RuleAttributes (attributeID → value). Return the Score
//     of the first variation whose conditions all pass.
//  3. No variation matched → return rule.Score.
func EvaluateRuleScore(rule entity.DecisionRule, userAttrs map[string]json.RawMessage) (*string, float64, error) {
	if len(rule.RuleConditions) == 0 {
		return nil, rule.Score, nil
	}

	// Check each Rule variation in OrderNo order; return first match's score.
	for _, v := range sortedVariations(rule.Rules) {
		// Build expected-value map: attributeID → value from this variation's RuleAttributes.
		expectedValues := make(map[string]json.RawMessage, len(v.RuleAttributes))
		for _, ra := range v.RuleAttributes {
			expectedValues[ra.AttributeID.String()] = json.RawMessage(ra.Value)
		}

		pass, err := evaluateConditionGroup(rule.RuleConditions, expectedValues, userAttrs)
		if err != nil {
			continue // Skip malformed variation rather than failing the whole rule.
		}
		if pass {
			return &v.VariationName, float64(v.Score), nil
		}
	}

	return nil, rule.Score, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// maxConditionDepth is the maximum nesting level supported by the condition tree.
const maxConditionDepth = 3

// evaluateConditionGroup evaluates a flat slice of conditions that may contain
// parent-child relationships (via ParentRuleConditionID). It builds a tree and
// evaluates it recursively up to maxConditionDepth levels.
//
// Tree structure:
//
//	Level 1 — root conditions (ParentRuleConditionID == nil)
//	Level 2 — children of level-1 nodes
//	Level 3 — children of level-2 nodes (leaf evaluation forced at this level)
func evaluateConditionGroup(conditions []entity.RuleCondition, expectedValues map[string]json.RawMessage, userAttrs map[string]json.RawMessage) (bool, error) {
	// Build parent → children index. Key "" represents root (no parent).
	byParent := make(map[string][]entity.RuleCondition, len(conditions))
	for _, c := range conditions {
		key := ""
		if c.ParentRuleConditionID != nil {
			key = c.ParentRuleConditionID.String()
		}
		byParent[key] = append(byParent[key], c)
	}
	// Sort each sibling group by Sequence.
	for k := range byParent {
		sort.Slice(byParent[k], func(i, j int) bool {
			return byParent[k][i].Sequence < byParent[k][j].Sequence
		})
	}
	roots := byParent[""]
	if len(roots) == 0 {
		return true, nil
	}
	return evalSiblings(byParent, roots, 1, expectedValues, userAttrs)
}

// evalSiblings evaluates a group of sibling conditions left-to-right, combining
// them with each condition's ConnectorOperator (AND by default, OR when specified).
// The first condition's ConnectorOperator is ignored.
func evalSiblings(byParent map[string][]entity.RuleCondition, siblings []entity.RuleCondition, depth int, expectedValues map[string]json.RawMessage, userAttrs map[string]json.RawMessage) (bool, error) {
	result, err := evalNode(byParent, siblings[0], depth, expectedValues, userAttrs)
	if err != nil {
		return false, err
	}
	for i := 1; i < len(siblings); i++ {
		c := siblings[i]
		val, err := evalNode(byParent, c, depth, expectedValues, userAttrs)
		if err != nil {
			return false, err
		}
		if c.ConnectorOperator == enums.ConnectorOperatorOR {
			result = result || val
		} else {
			result = result && val // AND is the default
		}
	}
	return result, nil
}

// evalNode evaluates one condition node.
// If the node has children and the depth limit has not been reached, it recurses
// into the children group (the node itself acts as a logical bracket).
// Otherwise it evaluates the condition's own attribute comparison directly.
func evalNode(byParent map[string][]entity.RuleCondition, c entity.RuleCondition, depth int, expectedValues map[string]json.RawMessage, userAttrs map[string]json.RawMessage) (bool, error) {
	if depth < maxConditionDepth {
		if children := byParent[c.ID.String()]; len(children) > 0 {
			return evalSiblings(byParent, children, depth+1, expectedValues, userAttrs)
		}
	}
	// Leaf node or depth limit reached — evaluate this condition directly.
	return evaluateSingleCondition(c, expectedValues, userAttrs)
}

// evaluateSingleCondition checks one RuleCondition against the user's live
// attribute values.
//
// The actual value is always read from userAttrs[attributeID].
// A nil userAttrs or a missing attribute key is treated as non-match (false, nil).
//
// Attribute.DataType (from the preloaded Attribute association) determines which
// type-specific comparator is used.
func evaluateSingleCondition(c entity.RuleCondition, expectedValues map[string]json.RawMessage, userAttrs map[string]json.RawMessage) (bool, error) {
	expectedRaw, ok := expectedValues[c.AttributeID.String()]
	if !ok {
		return false, nil // no expected value stamped — non-match
	}

	if userAttrs == nil {
		return false, nil // no user context — non-match
	}

	actualRaw, present := userAttrs[c.AttributeID.String()]
	if !present {
		return false, nil // missing attr = non-match
	}

	if c.Attribute == nil {
		return false, fmt.Errorf("condition %s: Attribute association not preloaded (need DataType)", c.ID)
	}

	return compareValues(
		c.Attribute.DataType,
		c.LogicalOperator,
		actualRaw,
		expectedRaw,
	)
}

// compareValues dispatches to the type-specific comparator.
func compareValues(
	dt enums.AttributeDataType,
	op enums.LogicalOperator,
	actualRaw, expectedRaw json.RawMessage,
) (bool, error) {
	switch dt {
	case enums.AttributeDataTypeText:
		return compareText(op, actualRaw, expectedRaw)
	case enums.AttributeDataTypeNumber:
		return compareNumber(op, actualRaw, expectedRaw)
	case enums.AttributeDataTypeDate:
		return compareDate(op, actualRaw, expectedRaw)
	case enums.AttributeDataTypeBoolean:
		return compareBoolean(op, actualRaw, expectedRaw)
	default:
		return false, fmt.Errorf("unsupported attribute data type %q", dt)
	}
}

// ---------------------------------------------------------------------------
// Per-type comparators
// ---------------------------------------------------------------------------

// compareText handles Text data type: supports =, !=, IN operators.
func compareText(op enums.LogicalOperator, actualRaw, expectedRaw json.RawMessage) (bool, error) {
	var actual string
	if err := json.Unmarshal(actualRaw, &actual); err != nil {
		return false, fmt.Errorf("parse text actual value: %w", err)
	}
	switch op {
	case enums.LogicalOperatorEQ:
		var expected string
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse text expected value: %w", err)
		}
		return actual == expected, nil
	case enums.LogicalOperatorNEQ:
		var expected string
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse text expected value: %w", err)
		}
		return actual != expected, nil
	case enums.LogicalOperatorIN:
		var expected []string
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse text IN values (want JSON string array): %w", err)
		}
		for _, v := range expected {
			if actual == v {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("operator %q not supported for Text attribute type", op)
	}
}

// compareNumber handles Number data type: supports =, !=, <, <=, >, >=, IN, BETWEEN operators.
//
//   - IN  — expectedRaw must be a JSON number array, e.g. [1,2,3]
//   - BETWEEN — expectedRaw must be a 2-element JSON array [min, max] (inclusive)
func compareNumber(op enums.LogicalOperator, actualRaw, expectedRaw json.RawMessage) (bool, error) {
	var actual float64
	if err := json.Unmarshal(actualRaw, &actual); err != nil {
		return false, fmt.Errorf("parse number actual value: %w", err)
	}
	switch op {
	case enums.LogicalOperatorEQ:
		var expected float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number expected value: %w", err)
		}
		return actual == expected, nil
	case enums.LogicalOperatorNEQ:
		var expected float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number expected value: %w", err)
		}
		return actual != expected, nil
	case enums.LogicalOperatorLT:
		var expected float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number expected value: %w", err)
		}
		return actual < expected, nil
	case enums.LogicalOperatorLTE:
		var expected float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number expected value: %w", err)
		}
		return actual <= expected, nil
	case enums.LogicalOperatorGT:
		var expected float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number expected value: %w", err)
		}
		return actual > expected, nil
	case enums.LogicalOperatorGTE:
		var expected float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number expected value: %w", err)
		}
		return actual >= expected, nil
	case enums.LogicalOperatorIN:
		var expected []float64
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse number IN values (want JSON number array): %w", err)
		}
		for _, v := range expected {
			if actual == v {
				return true, nil
			}
		}
		return false, nil
	case enums.LogicalOperatorBETWEEN:
		var bounds []float64
		if err := json.Unmarshal(expectedRaw, &bounds); err != nil {
			return false, fmt.Errorf("parse number BETWEEN bounds (want [min,max]): %w", err)
		}
		if len(bounds) != 2 {
			return false, fmt.Errorf("number BETWEEN expects exactly 2 bounds, got %d", len(bounds))
		}
		return actual >= bounds[0] && actual <= bounds[1], nil
	default:
		return false, fmt.Errorf("operator %q not supported for Number attribute type", op)
	}
}

// compareDate handles Date data type: supports =, !=, <, <=, >, >=, BETWEEN operators.
//
//   - Dates must be quoted JSON strings in RFC3339 or "YYYY-MM-DD" format.
//   - BETWEEN — expectedRaw must be a 2-element JSON string array ["from","to"] (inclusive).
func compareDate(op enums.LogicalOperator, actualRaw, expectedRaw json.RawMessage) (bool, error) {
	var actualStr string
	if err := json.Unmarshal(actualRaw, &actualStr); err != nil {
		return false, fmt.Errorf("parse date actual value: %w", err)
	}
	actual, err := parseDate(actualStr)
	if err != nil {
		return false, fmt.Errorf("date actual: %w", err)
	}

	switch op {
	case enums.LogicalOperatorEQ:
		var s string
		if err := json.Unmarshal(expectedRaw, &s); err != nil {
			return false, fmt.Errorf("parse date expected value: %w", err)
		}
		expected, err := parseDate(s)
		if err != nil {
			return false, fmt.Errorf("date expected: %w", err)
		}
		return actual.Equal(expected), nil
	case enums.LogicalOperatorNEQ:
		var s string
		if err := json.Unmarshal(expectedRaw, &s); err != nil {
			return false, fmt.Errorf("parse date expected value: %w", err)
		}
		expected, err := parseDate(s)
		if err != nil {
			return false, fmt.Errorf("date expected: %w", err)
		}
		return !actual.Equal(expected), nil
	case enums.LogicalOperatorLT:
		var s string
		if err := json.Unmarshal(expectedRaw, &s); err != nil {
			return false, fmt.Errorf("parse date expected value: %w", err)
		}
		expected, err := parseDate(s)
		if err != nil {
			return false, fmt.Errorf("date expected: %w", err)
		}
		return actual.Before(expected), nil
	case enums.LogicalOperatorLTE:
		var s string
		if err := json.Unmarshal(expectedRaw, &s); err != nil {
			return false, fmt.Errorf("parse date expected value: %w", err)
		}
		expected, err := parseDate(s)
		if err != nil {
			return false, fmt.Errorf("date expected: %w", err)
		}
		return actual.Before(expected) || actual.Equal(expected), nil
	case enums.LogicalOperatorGT:
		var s string
		if err := json.Unmarshal(expectedRaw, &s); err != nil {
			return false, fmt.Errorf("parse date expected value: %w", err)
		}
		expected, err := parseDate(s)
		if err != nil {
			return false, fmt.Errorf("date expected: %w", err)
		}
		return actual.After(expected), nil
	case enums.LogicalOperatorGTE:
		var s string
		if err := json.Unmarshal(expectedRaw, &s); err != nil {
			return false, fmt.Errorf("parse date expected value: %w", err)
		}
		expected, err := parseDate(s)
		if err != nil {
			return false, fmt.Errorf("date expected: %w", err)
		}
		return actual.After(expected) || actual.Equal(expected), nil
	case enums.LogicalOperatorBETWEEN:
		var bounds []string
		if err := json.Unmarshal(expectedRaw, &bounds); err != nil {
			return false, fmt.Errorf("parse date BETWEEN bounds (want [\"from\",\"to\"]): %w", err)
		}
		if len(bounds) != 2 {
			return false, fmt.Errorf("date BETWEEN expects exactly 2 bounds, got %d", len(bounds))
		}
		lo, err := parseDate(bounds[0])
		if err != nil {
			return false, fmt.Errorf("date BETWEEN lower: %w", err)
		}
		hi, err := parseDate(bounds[1])
		if err != nil {
			return false, fmt.Errorf("date BETWEEN upper: %w", err)
		}
		return (actual.Equal(lo) || actual.After(lo)) && (actual.Equal(hi) || actual.Before(hi)), nil
	default:
		return false, fmt.Errorf("operator %q not supported for Date attribute type", op)
	}
}

// compareBoolean handles Boolean data type: supports = and != operators.
func compareBoolean(op enums.LogicalOperator, actualRaw, expectedRaw json.RawMessage) (bool, error) {
	var actual bool
	if err := json.Unmarshal(actualRaw, &actual); err != nil {
		return false, fmt.Errorf("parse boolean actual value: %w", err)
	}
	switch op {
	case enums.LogicalOperatorEQ:
		var expected bool
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse boolean expected value: %w", err)
		}
		return actual == expected, nil
	case enums.LogicalOperatorNEQ:
		var expected bool
		if err := json.Unmarshal(expectedRaw, &expected); err != nil {
			return false, fmt.Errorf("parse boolean expected value: %w", err)
		}
		return actual != expected, nil
	default:
		return false, fmt.Errorf("operator %q not supported for Boolean attribute type", op)
	}
}

// logicConditionToRuleCondition converts a stored LogicCondition into a minimal
// entity.RuleCondition stub used for tree traversal and leaf evaluation via the
// unified evaluateSingleCondition path.
//
// The Attribute stub carries only DataType (needed by compareValues).
// EvaluateLogicConditions always passes a non-nil userAttrs map, so the actual
// comparison value comes from there — never from the Attribute entity.
func logicConditionToRuleCondition(lc domainservice.LogicCondition) entity.RuleCondition {
	id, _ := uuid.Parse(lc.ConditionID)
	rc := entity.RuleCondition{
		BaseModel:         entity.BaseModel{ID: id},
		AttributeID:       mustParseUUID(lc.AttributeID),
		Sequence:          lc.Sequence,
		LogicalOperator:   enums.LogicalOperator(lc.LogicalOperator),
		ConnectorOperator: enums.ConnectorOperator(lc.ConnectorOperator),
		Attribute:         &entity.Attribute{DataType: enums.AttributeDataType(lc.DataType)},
	}
	if lc.ParentConditionID != "" {
		pid, _ := uuid.Parse(lc.ParentConditionID)
		rc.ParentRuleConditionID = &pid
	}
	return rc
}

// mustParseUUID parses a UUID string and returns uuid.Nil on error.
func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

// EvaluateLogicConditions evaluates a slice of LogicCondition (from PlacementLogicEntry)
// against live user attribute values supplied in userAttrs (attr UUID → compact JSON value).
//
// It converts the LogicCondition slice into entity.RuleCondition stubs (including an
// Attribute stub that carries the DataType) and delegates to the unified
// evaluateConditionGroup chain. The "actual" value for each leaf is read from userAttrs;
// the "expected" value is taken from LogicCondition.ExpectedValue.
//
// Returns false (not an error) when a required attribute is absent from userAttrs.
func EvaluateLogicConditions(conditions []domainservice.LogicCondition, userAttrs map[string]json.RawMessage) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	// Convert LogicCondition → entity.RuleCondition stubs (includes Attribute.DataType stub).
	rcs := make([]entity.RuleCondition, 0, len(conditions))
	for _, lc := range conditions {
		rcs = append(rcs, logicConditionToRuleCondition(lc))
	}

	// Build expectedValues map: attributeID → LogicCondition.ExpectedValue.
	// Only stamp entries where ExpectedValue is a non-nil, non-JSON-null value.
	// Conditions without a stamped expected value (e.g. parent/grouping nodes or
	// missing rule_attribute rows) must be skipped so that the !ok guard in
	// evaluateSingleCondition returns non-match rather than comparing against a
	// zero-value produced by unmarshalling JSON null.
	expectedValues := make(map[string]json.RawMessage, len(conditions))
	for _, lc := range conditions {
		if len(lc.ExpectedValue) > 0 && string(lc.ExpectedValue) != "null" {
			expectedValues[lc.AttributeID] = lc.ExpectedValue
		}
	}

	// Delegate to the unified chain; userAttrs provides the actual values.
	return evaluateConditionGroup(rcs, expectedValues, userAttrs)
}

// ---------------------------------------------------------------------------
// Sorting helpers
// ---------------------------------------------------------------------------

func sortedVariations(rules []entity.Rule) []entity.Rule {
	out := make([]entity.Rule, len(rules))
	copy(out, rules)
	sort.Slice(out, func(i, j int) bool {
		return out[i].OrderNo < out[j].OrderNo
	})
	return out
}

// parseDate tries RFC3339 then "YYYY-MM-DD" date-only format.
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q (expected RFC3339 or YYYY-MM-DD)", s)
}
