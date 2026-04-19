package enums

import "fmt"

// CalendarType represents the category of a calendar source.
type CalendarType string

const (
	CalendarTypeHoliday  CalendarType = "HOLIDAY"
	CalendarTypePersonal CalendarType = "PERSONAL"
	CalendarTypeCustom   CalendarType = "CUSTOM"
)

// String returns the string representation of the CalendarType.
func (c CalendarType) String() string {
	return string(c)
}

// IsValid reports whether c is a known CalendarType constant.
func (c CalendarType) IsValid() bool {
	switch c {
	case CalendarTypeHoliday, CalendarTypePersonal, CalendarTypeCustom:
		return true
	}
	return false
}

// Parse parses a raw string into a CalendarType.
// Returns an error if the value is not a valid type.
func (c CalendarType) Parse(val string) (CalendarType, error) {
	v := CalendarType(val)
	if !v.IsValid() {
		return "", fmt.Errorf("invalid CalendarType %q: must be one of HOLIDAY, PERSONAL, CUSTOM", val)
	}
	return v, nil
}
