package entity

import (
	"time"

	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
)

// ScheduleOccurrence stores a materialized instance of a Schedule.
//
// Table: schedule_occurrences
type ScheduleOccurrence struct {
	BaseModel
	ScheduleID      uuid.UUID              `gorm:"type:uuid;not null"    json:"scheduleId"`
	Schedule        *Schedule              `gorm:"foreignKey:ScheduleID" json:"schedule,omitempty"`
	OccurrenceStart time.Time              `gorm:"type:timestamptz"      json:"occurrenceStart"`
	OccurrenceEnd   time.Time              `gorm:"type:timestamptz"      json:"occurrenceEnd"`
	Status          enums.OccurrenceStatus `gorm:"size:255"              json:"status"`
	Source          enums.OccurrenceSource `gorm:"size:255"              json:"source"`
}
