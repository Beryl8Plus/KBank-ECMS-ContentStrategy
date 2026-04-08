package entity

import (
	"time"

	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
)

// Schedule links a DecisionRule to a Placement with recurrence-based scheduling.
//
// Table: schedules
type Schedule struct {
	BaseModel
	DecisionRuleID uuid.UUID            `gorm:"type:uuid"                       json:"decisionRuleId"`
	DecisionRule   *DecisionRule        `gorm:"foreignKey:DecisionRuleID"       json:"decisionRule,omitempty"`
	PlacementID    uuid.UUID            `gorm:"type:uuid"                       json:"placementId"`
	Placement      *Placement           `gorm:"foreignKey:PlacementID"          json:"placement,omitempty"`
	CalendarID     *uuid.UUID           `gorm:"type:uuid"                       json:"calendarId"`
	Calendar       *Calendar            `gorm:"foreignKey:CalendarID"           json:"calendar,omitempty"`
	RecurrenceType enums.RecurrenceType `gorm:"size:255"                        json:"recurrenceType"`
	RecurrenceRule *string              `gorm:"type:text"                       json:"recurrenceRule"`
	EffectiveFrom  time.Time            `gorm:"type:timestamptz"                json:"effectiveFrom"`
	EffectiveUntil time.Time            `gorm:"type:timestamptz"                json:"effectiveUntil"`
	TimeOfDayStart *string              `gorm:"size:5"                          json:"timeOfDayStart"`
	TimeOfDayEnd   *string              `gorm:"size:5"                          json:"timeOfDayEnd"`
	AllDay         bool                 `gorm:"default:false"                   json:"allDay"`
	Timezone       string               `gorm:"size:255;default:'Asia/Bangkok'" json:"timezone"`
	IsActive       bool                 `gorm:"default:false"                   json:"isActive"`
}
