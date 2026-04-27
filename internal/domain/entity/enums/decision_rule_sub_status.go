package enums

import "fmt"

// DecisionRuleSubStatus provides finer-grained lifecycle detail for a DecisionRule.
type DecisionRuleSubStatus string

const (
	DecisionRuleSubStatusNA      DecisionRuleSubStatus = "N/A"
	DecisionRuleSubStatusMissing DecisionRuleSubStatus = "Missing attribute registry"
)

func (s DecisionRuleSubStatus) String() string {
	return string(s)
}

func (s DecisionRuleSubStatus) IsValid() bool {
	switch s {
	case DecisionRuleSubStatusNA, DecisionRuleSubStatusMissing:
		return true
	}
	return false
}

func (s DecisionRuleSubStatus) Parse(val string) (DecisionRuleSubStatus, error) {
	v := DecisionRuleSubStatus(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid DecisionRuleSubStatus %q: must be one of N/A, Missing attribute registry", val)
	}
	return v, nil
}
