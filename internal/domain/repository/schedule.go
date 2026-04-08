package repository

import (
	"context"
	"time"

	"kbank-ecms/internal/domain/entity"

	"github.com/google/uuid"
)

// ScheduleRepository defines the contract for schedule-related database operations.
type ScheduleRepository interface {
	// CheckScheduleOverlap returns the first active non-deleted schedule for the same
	// (decision_rule_id, placement_id) pair whose [effective_from, effective_until) range
	// overlaps [effectiveFrom, effectiveUntil).
	// If excludeID is non-nil, that schedule is excluded from the check (for self-update scenarios).
	// Returns (nil, nil) when no conflicting schedule is found.
	CheckScheduleOverlap(
		ctx context.Context,
		decisionRuleID uuid.UUID,
		placementID uuid.UUID,
		effectiveFrom, effectiveUntil time.Time,
		excludeID *uuid.UUID,
	) (*entity.Schedule, error)

	// CreateSchedule persists a new schedule record.
	CreateSchedule(ctx context.Context, schedule *entity.Schedule) error

	// GetScheduleByID retrieves a single non-deleted schedule by its primary key.
	// Returns (nil, nil) when no record is found.
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)

	// ListSchedules returns all non-deleted schedules ordered by created_at descending.
	ListSchedules(ctx context.Context) ([]*entity.Schedule, error)

	// ListSchedulesPaginated returns a page of non-deleted schedules ordered by
	// created_at descending together with the total count of matching records.
	ListSchedulesPaginated(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error)

	// UpdateSchedule saves all fields of an existing schedule record.
	UpdateSchedule(ctx context.Context, schedule *entity.Schedule) error

	// DeleteSchedule soft-deletes the schedule with the given ID.
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
}
