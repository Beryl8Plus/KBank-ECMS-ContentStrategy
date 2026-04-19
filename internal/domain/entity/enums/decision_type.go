package enums

import (
	"fmt"
	"slices"
)

type DecisionType string

const (
	DecisionTypeMass        DecisionType = "MASS"
	DecisionTypeAudience    DecisionType = "AUDIENCE"
	DecisionTypeSalesTarget DecisionType = "SALES_TARGET"
	DecisionTypeNonSales    DecisionType = "NON_SALES"
)

var decisionTypes = []DecisionType{
	DecisionTypeMass,
	DecisionTypeAudience,
	DecisionTypeSalesTarget,
	DecisionTypeNonSales,
}

// String returns the string representation of the DecisionType.
func (f DecisionType) String() string {
	return string(f)
}

// IsValid checks if the DecisionType is valid.
func (f DecisionType) IsValid() bool {
	return slices.Contains(decisionTypes, f)
}

// Values returns all valid DecisionType values.
func (f DecisionType) Values() []DecisionType {
	return decisionTypes
}

// ParseDecisionType parses a string into a DecisionType.
func (f DecisionType) Parse(val string) (DecisionType, error) {
	for _, v := range decisionTypes {
		if v.String() == val {
			return v, nil
		}
	}
	return "", fmt.Errorf("invalid feature code: %s", val)
}

func (f DecisionType) IsCampaign() bool {
	return f == DecisionTypeAudience || f == DecisionTypeSalesTarget
}
