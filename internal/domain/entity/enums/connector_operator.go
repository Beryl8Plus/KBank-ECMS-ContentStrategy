package enums

import "fmt"

// ConnectorOperator represents a logical connector between adjacent rule conditions.
type ConnectorOperator string

const (
	ConnectorOperatorAND ConnectorOperator = "AND"
	ConnectorOperatorOR  ConnectorOperator = "OR"
)

// String returns the string representation of the ConnectorOperator.
func (o ConnectorOperator) String() string {
	return string(o)
}

// IsValid reports whether o is a known ConnectorOperator constant.
func (o ConnectorOperator) IsValid() bool {
	switch o {
	case ConnectorOperatorAND, ConnectorOperatorOR:
		return true
	}
	return false
}

// ParseConnectorOperator parses a raw string into a ConnectorOperator.
// Returns an error if the value is not a valid connector.
func ParseConnectorOperator(s string) (ConnectorOperator, error) {
	v := ConnectorOperator(s)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid ConnectorOperator %q: must be one of AND, OR", s)
	}
	return v, nil
}
