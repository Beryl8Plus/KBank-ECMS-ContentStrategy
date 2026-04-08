package enums

import "fmt"

// LogicalOperator represents a comparison operator used in rule conditions.
type LogicalOperator string

const (
	LogicalOperatorLT      LogicalOperator = "<"
	LogicalOperatorGT      LogicalOperator = ">"
	LogicalOperatorEQ      LogicalOperator = "="
	LogicalOperatorIN      LogicalOperator = "IN"
	LogicalOperatorBETWEEN LogicalOperator = "BETWEEN"
)

// String returns the string representation of the LogicalOperator.
func (o LogicalOperator) String() string {
	return string(o)
}

// IsValid reports whether o is a known LogicalOperator constant.
func (o LogicalOperator) IsValid() bool {
	switch o {
	case LogicalOperatorLT, LogicalOperatorGT, LogicalOperatorEQ, LogicalOperatorIN, LogicalOperatorBETWEEN:
		return true
	}
	return false
}

// ParseLogicalOperator parses a raw string into a LogicalOperator.
// Returns an error if the value is not a valid operator.
func ParseLogicalOperator(s string) (LogicalOperator, error) {
	v := LogicalOperator(s)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid LogicalOperator %q: must be one of <, >, =, IN, BETWEEN", s)
	}
	return v, nil
}
