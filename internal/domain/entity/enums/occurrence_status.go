package enums

import "fmt"

// OccurrenceStatus represents the status of a schedule occurrence instance.
type OccurrenceStatus string

const (
	OccurrenceStatusActive    OccurrenceStatus = "ACTIVE"
	OccurrenceStatusCancelled OccurrenceStatus = "CANCELLED"
	OccurrenceStatusModified  OccurrenceStatus = "MODIFIED"
)

// String returns the string representation of the OccurrenceStatus.
func (o OccurrenceStatus) String() string {
	return string(o)
}

// IsValid reports whether o is a known OccurrenceStatus constant.
func (o OccurrenceStatus) IsValid() bool {
	switch o {
	case OccurrenceStatusActive, OccurrenceStatusCancelled, OccurrenceStatusModified:
		return true
	}
	return false
}

// Parse parses a raw string into an OccurrenceStatus.
// Returns an error if the value is not a valid status.
func (o OccurrenceStatus) Parse(s string) (OccurrenceStatus, error) {
	v := OccurrenceStatus(s)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid OccurrenceStatus %q: must be one of ACTIVE, CANCELLED, MODIFIED", s)
	}
	return v, nil
}
