package enums

import "fmt"

// AttributeDataType represents the value category of an attribute used in rule conditions.
type AttributeDataType string

const (
	AttributeDataTypeText    AttributeDataType = "Text"
	AttributeDataTypeDate    AttributeDataType = "Date"
	AttributeDataTypeNumber  AttributeDataType = "Number"
	AttributeDataTypeBoolean AttributeDataType = "Boolean"
)

// String returns the string representation of the AttributeDataType.
func (t AttributeDataType) String() string {
	return string(t)
}

// IsValid reports whether t is a known AttributeDataType constant.
func (t AttributeDataType) IsValid() bool {
	switch t {
	case AttributeDataTypeText, AttributeDataTypeDate, AttributeDataTypeNumber, AttributeDataTypeBoolean:
		return true
	}
	return false
}

// Parse parses a raw string into an AttributeDataType.
// Returns an error if the value is not a valid data type.
func (t AttributeDataType) Parse(val string) (AttributeDataType, error) {
	v := AttributeDataType(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid AttributeDataType %q: must be one of Text, Date, Number, Boolean", val)
	}
	return v, nil
}
