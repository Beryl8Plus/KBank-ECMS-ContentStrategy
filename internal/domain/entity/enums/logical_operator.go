package enums

import "fmt"

// LogicalOperator represents a comparison operator used in rule conditions.
type LogicalOperator string

const (
	LogicalOperatorLT       LogicalOperator = "<"
	LogicalOperatorLTE      LogicalOperator = "<="
	LogicalOperatorGT       LogicalOperator = ">"
	LogicalOperatorGTE      LogicalOperator = ">="
	LogicalOperatorEQ       LogicalOperator = "="
	LogicalOperatorNEQ      LogicalOperator = "!="
	LogicalOperatorIN       LogicalOperator = "IN"
	LogicalOperatorNIN      LogicalOperator = "NOT IN"
	LogicalOperatorBETWEEN  LogicalOperator = "BETWEEN"
	LogicalOperatorCONTAINS LogicalOperator = "CONTAINS"
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
		LogicalOperatorIN, LogicalOperatorNIN,
		LogicalOperatorBETWEEN, LogicalOperatorCONTAINS:
		return true
	}
	return false
}

// GetAllOperators returns a slice of all defined LogicalOperator constants.
func (o LogicalOperator) GetAllOperators() []LogicalOperator {
	return []LogicalOperator{
		LogicalOperatorLT, LogicalOperatorLTE,
		LogicalOperatorGT, LogicalOperatorGTE,
		LogicalOperatorEQ, LogicalOperatorNEQ,
		LogicalOperatorIN, LogicalOperatorNIN,
		LogicalOperatorBETWEEN, LogicalOperatorCONTAINS,
	}
}

// Parse parses a raw string into a LogicalOperator.
// Returns an error if the value is not a valid operator.
func (o LogicalOperator) Parse(val string) (LogicalOperator, error) {
	v := LogicalOperator(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid LogicalOperator %q: must be one of %v", val, v.GetAllOperators())
	}
	return v, nil
}
