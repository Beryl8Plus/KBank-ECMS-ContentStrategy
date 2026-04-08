package dto

import (
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// CreateScheduleRequest is the request body for POST /schedules.
type CreateScheduleRequest struct {
	DecisionRuleID uuid.UUID            `json:"decisionRuleId" binding:"required"`
	PlacementID    uuid.UUID            `json:"placementId" binding:"required"`
	CalendarID     *uuid.UUID           `json:"calendarId"`
	RecurrenceType enums.RecurrenceType `json:"recurrenceType" binding:"required"`
	RecurrenceRule *string              `json:"recurrenceRule"`
	EffectiveFrom  time.Time            `json:"effectiveFrom" binding:"required"`
	EffectiveUntil time.Time            `json:"effectiveUntil" binding:"required"`
	TimeOfDayStart *string              `json:"timeOfDayStart"`
	TimeOfDayEnd   *string              `json:"timeOfDayEnd"`
	AllDay         bool                 `json:"allDay"`
	Timezone       string               `json:"timezone"`
	IsActive       bool                 `json:"isActive"`
}

// UpdateScheduleRequest is the request body for PUT /schedules/:id.
// DecisionRuleID and PlacementID are immutable after creation.
type UpdateScheduleRequest struct {
	CalendarID     *uuid.UUID           `json:"calendarId"`
	RecurrenceType enums.RecurrenceType `json:"recurrenceType" binding:"required"`
	RecurrenceRule *string              `json:"recurrenceRule"`
	EffectiveFrom  time.Time            `json:"effectiveFrom" binding:"required"`
	EffectiveUntil time.Time            `json:"effectiveUntil" binding:"required"`
	TimeOfDayStart *string              `json:"timeOfDayStart"`
	TimeOfDayEnd   *string              `json:"timeOfDayEnd"`
	AllDay         bool                 `json:"allDay"`
	Timezone       string               `json:"timezone"`
	IsActive       bool                 `json:"isActive"`
}

// ScheduleResponse is the response body for schedule endpoints.
type ScheduleResponse struct {
	ID             uuid.UUID            `json:"id"`
	DecisionRuleID uuid.UUID            `json:"decisionRuleId"`
	PlacementID    uuid.UUID            `json:"placementId"`
	CalendarID     *uuid.UUID           `json:"calendarId"`
	RecurrenceType enums.RecurrenceType `json:"recurrenceType"`
	RecurrenceRule *string              `json:"recurrenceRule"`
	EffectiveFrom  time.Time            `json:"effectiveFrom"`
	EffectiveUntil time.Time            `json:"effectiveUntil"`
	TimeOfDayStart *string              `json:"timeOfDayStart"`
	TimeOfDayEnd   *string              `json:"timeOfDayEnd"`
	AllDay         bool                 `json:"allDay"`
	Timezone       string               `json:"timezone"`
	IsActive       bool                 `json:"isActive"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

// ToScheduleResponse converts a Schedule entity to a ScheduleResponse DTO.
func ToScheduleResponse(s *entity.Schedule) ScheduleResponse {
	return ScheduleResponse{
		ID:             s.ID,
		DecisionRuleID: s.DecisionRuleID,
		PlacementID:    s.PlacementID,
		CalendarID:     s.CalendarID,
		RecurrenceType: s.RecurrenceType,
		RecurrenceRule: s.RecurrenceRule,
		EffectiveFrom:  s.EffectiveFrom,
		EffectiveUntil: s.EffectiveUntil,
		TimeOfDayStart: s.TimeOfDayStart,
		TimeOfDayEnd:   s.TimeOfDayEnd,
		AllDay:         s.AllDay,
		Timezone:       s.Timezone,
		IsActive:       s.IsActive,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}
