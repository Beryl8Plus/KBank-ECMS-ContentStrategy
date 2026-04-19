package enums

import "fmt"

// LogicalOperator represents a comparison operator used in rule conditions.
type LogicalOperator string

const (
	LogicalOperatorLT      LogicalOperator = "<"
	LogicalOperatorLTE     LogicalOperator = "<="
	LogicalOperatorGT      LogicalOperator = ">"
	LogicalOperatorGTE     LogicalOperator = ">="
	LogicalOperatorEQ      LogicalOperator = "="
	LogicalOperatorNEQ     LogicalOperator = "!="
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
	case
		LogicalOperatorLT, LogicalOperatorLTE,
		LogicalOperatorGT, LogicalOperatorGTE,
		LogicalOperatorEQ, LogicalOperatorNEQ,
		LogicalOperatorIN, LogicalOperatorBETWEEN:
		return true
	}
	return false
}

// Parse parses a raw string into a LogicalOperator.
// Returns an error if the value is not a valid operator.
func (o LogicalOperator) Parse(val string) (LogicalOperator, error) {
	v := LogicalOperator(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid LogicalOperator %q: must be one of <, <=, >, >=, =, !=, IN, BETWEEN", val)
	}
	return v, nil
}
