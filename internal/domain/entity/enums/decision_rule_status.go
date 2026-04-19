package enums

import "fmt"

// DecisionRuleStatus represents the lifecycle status of a decision rule.
// Valid values are:
//   - DRAFT: The rule is being created or modified and is not yet active.
//   - ACTIVE: The rule is active and can be evaluated.
//   - INACTIVE: The rule is inactive and will not be evaluated.
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

// Parse parses a raw string into a DecisionRuleStatus.
// Returns an error if the value is not a valid status.
func (s DecisionRuleStatus) Parse(val string) (DecisionRuleStatus, error) {
	v := DecisionRuleStatus(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid DecisionRuleStatus %q: must be one of DRAFT, ACTIVE, INACTIVE", s)
	}
	return v, nil
}
