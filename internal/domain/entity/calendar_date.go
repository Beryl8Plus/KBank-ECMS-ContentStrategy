package entity

import (
	"time"

	"github.com/google/uuid"
)

// CalendarDate stores an individual date entry within a Calendar.
//
// Table: calendar_dates
type CalendarDate struct {
	BaseModel
	CalendarID  uuid.UUID `gorm:"type:uuid;not null"    json:"calendarId"`
	Calendar    *Calendar `gorm:"foreignKey:CalendarID" json:"calendar,omitempty"`
	Date        time.Time `gorm:"type:date"             json:"date"`
	Name        string    `gorm:"size:255"              json:"name"`
	IsRecurring bool      `gorm:"default:false"         json:"isRecurring"`
}
