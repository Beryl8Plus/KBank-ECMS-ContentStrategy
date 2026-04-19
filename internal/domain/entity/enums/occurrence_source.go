package enums

import "fmt"

// OccurrenceSource represents how a schedule occurrence was generated.
type OccurrenceSource string

const (
	OccurrenceSourceRecurrence OccurrenceSource = "RECURRENCE"
	OccurrenceSourceCalendar   OccurrenceSource = "CALENDAR"
	OccurrenceSourceManual     OccurrenceSource = "MANUAL"
)

// String returns the string representation of the OccurrenceSource.
func (o OccurrenceSource) String() string {
	return string(o)
}

// IsValid reports whether o is a known OccurrenceSource constant.
func (o OccurrenceSource) IsValid() bool {
	switch o {
	case OccurrenceSourceRecurrence, OccurrenceSourceCalendar, OccurrenceSourceManual:
		return true
	}
	return false
}

// Parse parses a raw string into an OccurrenceSource.
// Returns an error if the value is not a valid source.
func (o OccurrenceSource) Parse(val string) (OccurrenceSource, error) {
	v := OccurrenceSource(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid OccurrenceSource %q: must be one of RECURRENCE, CALENDAR, MANUAL", val)
	}
	return v, nil
}
