package enums

import (
	"fmt"
	"slices"
)

type Feature string

const (
	FeatureContentDecisionRule Feature = "CONTENT_DECISION_RULE"
	// Add new feature code here
)

var featureCodes = []Feature{
	FeatureContentDecisionRule,
}

// String returns the string representation of the Feature.
func (f Feature) String() string {
	return string(f)
}

// IsValid checks if the Feature is valid.
func (f Feature) IsValid() bool {
	return slices.Contains(featureCodes, f)
}

// Values returns all valid Feature values.
func Values() []Feature {
	return featureCodes
}

// ParseFeature parses a string into a Feature.
func ParseFeature(s string) (Feature, error) {
	for _, v := range featureCodes {
		if v.String() == s {
			return v, nil
		}
	}
	return "", fmt.Errorf("invalid feature code: %s", s)
}
