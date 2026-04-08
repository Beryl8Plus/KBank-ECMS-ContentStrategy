package enums

import "fmt"

// DecisionRuleType represents the category of a decision rule.
type DecisionRuleType string

const (
	DecisionRuleTypeScoring  DecisionRuleType = "SCORING"  // การให้คะแนน
	DecisionRuleTypeSegment  DecisionRuleType = "SEGMENT"  // การแบ่งกลุ่ม
	DecisionRuleTypeEligible DecisionRuleType = "ELIGIBLE" // การคัดกรอง
)

// String returns the string representation of the DecisionRuleType.
func (t DecisionRuleType) String() string {
	return string(t)
}

// IsValid reports whether t is a known DecisionRuleType constant.
func (t DecisionRuleType) IsValid() bool {
	switch t {
	case DecisionRuleTypeScoring, DecisionRuleTypeSegment, DecisionRuleTypeEligible:
		return true
	}
	return false
}

// ParseDecisionRuleType parses a raw string into a DecisionRuleType.
// Returns an error if the value is not a valid type.
func ParseDecisionRuleType(s string) (DecisionRuleType, error) {
	v := DecisionRuleType(s)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid DecisionRuleType %q: must be one of SCORING, SEGMENT, ELIGIBLE", s)
	}
	return v, nil
}
