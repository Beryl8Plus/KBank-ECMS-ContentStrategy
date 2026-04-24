package enums

import "fmt"

type DecisionRuleSubStatus string

const (
	DecisionRuleSubStatusNone    DecisionRuleSubStatus = "N/A"
	DicisionRuleSubStatusMissing DecisionRuleSubStatus = "Missing attribute registry"
)

// String returns the string representation of the DecisionRuleSubStatus.
func (s DecisionRuleSubStatus) String() string {
	return string(s)
}

// IsValid reports whether s is a known DecisionRuleSubStatus constant.
func (s DecisionRuleSubStatus) IsValid() bool {
	switch s {
	case DecisionRuleSubStatusNone, DicisionRuleSubStatusMissing:
		return true
	}
	return false
}

// Parse parses a raw string into a DecisionRuleSubStatus.
// Returns an error if the value is not a valid status.
func (s DecisionRuleSubStatus) Parse(val string) (DecisionRuleStatus, error) {
	v := DecisionRuleStatus(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid DecisionRuleSubStatus %q: must be one of N/A, Missing attribute registry", s)
	}
	return v, nil
}
