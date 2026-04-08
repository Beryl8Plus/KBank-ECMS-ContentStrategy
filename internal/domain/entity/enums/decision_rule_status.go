package enums

import "fmt"

// DecisionRuleStatus represents the lifecycle status of a decision rule.
type DecisionRuleStatus string

const (
	DecisionRuleStatusDraft    DecisionRuleStatus = "DRAFT"
	DecisionRuleStatusActive   DecisionRuleStatus = "ACTIVE"
	DecisionRuleStatusInactive DecisionRuleStatus = "INACTIVE"
)

// String returns the string representation of the DecisionRuleStatus.
func (s DecisionRuleStatus) String() string {
	return string(s)
}

// IsValid reports whether s is a known DecisionRuleStatus constant.
func (s DecisionRuleStatus) IsValid() bool {
	switch s {
	case DecisionRuleStatusDraft, DecisionRuleStatusActive, DecisionRuleStatusInactive:
		return true
	}
	return false
}

// ParseDecisionRuleStatus parses a raw string into a DecisionRuleStatus.
// Returns an error if the value is not a valid status.
func ParseDecisionRuleStatus(s string) (DecisionRuleStatus, error) {
	v := DecisionRuleStatus(s)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid DecisionRuleStatus %q: must be one of DRAFT, ACTIVE, INACTIVE", s)
	}
	return v, nil
}
