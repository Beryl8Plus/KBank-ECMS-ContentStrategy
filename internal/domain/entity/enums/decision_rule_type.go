package enums

import "fmt"

// EvaluateType represents the category of a decision rule.
type EvaluateType string

const (
	EvaluateTypeScoring  EvaluateType = "SCORING"  // การให้คะแนน
	EvaluateTypeSegment  EvaluateType = "SEGMENT"  // การแบ่งกลุ่ม
	EvaluateTypeEligible EvaluateType = "ELIGIBLE" // การคัดกรอง
)

// String returns the string representation of the EvaluateType.
func (t EvaluateType) String() string {
	return string(t)
}

// IsValid reports whether t is a known EvaluateType constant.
func (t EvaluateType) IsValid() bool {
	switch t {
	case EvaluateTypeScoring, EvaluateTypeSegment, EvaluateTypeEligible:
		return true
	}
	return false
}

// Parse parses a raw string into a EvaluateType.
// Returns an error if the value is not a valid type.
func (t EvaluateType) Parse(val string) (EvaluateType, error) {
	v := EvaluateType(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid EvaluateType %q: must be one of SCORING, SEGMENT, ELIGIBLE", val)
	}
	return v, nil
}
