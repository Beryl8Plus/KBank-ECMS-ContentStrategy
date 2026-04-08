package enums

import "fmt"

// RecurrenceType represents the scheduling recurrence strategy.
type RecurrenceType string

const (
	RecurrenceTypeOnce     RecurrenceType = "ONCE"     // Occurs only once at the specified time.
	RecurrenceTypeRRule    RecurrenceType = "RRULE"    // Uses iCalendar RRULE syntax for complex recurring patterns.
	RecurrenceTypeCalendar RecurrenceType = "CALENDAR" // Follows predefined calendar events (e.g., public holidays, weekends).
)

// String returns the string representation of the RecurrenceType.
func (r RecurrenceType) String() string {
	return string(r)
}

// IsValid reports whether r is a known RecurrenceType constant.
func (r RecurrenceType) IsValid() bool {
	switch r {
	case RecurrenceTypeOnce, RecurrenceTypeRRule, RecurrenceTypeCalendar:
		return true
	}
	return false
}

// ParseRecurrenceType parses a raw string into a RecurrenceType.
// Returns an error if the value is not a valid type.
func ParseRecurrenceType(s string) (RecurrenceType, error) {
	v := RecurrenceType(s)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid RecurrenceType %q: must be one of ONCE, RRULE, CALENDAR", s)
	}
	return v, nil
}
