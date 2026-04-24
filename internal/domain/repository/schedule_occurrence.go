package repository

import (
	"context"
	"time"

	"kbank-ecms/internal/domain/entity"

	"github.com/google/uuid"
)

// ScheduleOccurrenceRepository defines the contract for schedule occurrence
// database operations. Occurrences are the materialized, pre-computed time
// windows derived from active Schedule records.
type ScheduleOccurrenceRepository interface {
	// UpsertOccurrences inserts or updates a batch of schedule occurrences.
	// Idempotency is guaranteed by the unique constraint on
	// (schedule_id, occurrence_start, occurrence_end): duplicates are ignored.
	UpsertOccurrences(ctx context.Context, occurrences []*entity.ScheduleOccurrence) error

	// DeleteFutureByScheduleID removes all occurrences for a given schedule
	// whose occurrence_start is after `after`. This is called when a Schedule
	// is updated or deleted so stale future occurrences are cleared before
	// re-materialisation.
	DeleteFutureByScheduleID(ctx context.Context, scheduleID uuid.UUID, after time.Time) error

	// DeletePastOccurrences removes all occurrences whose occurrence_end is
	// before `before`. Used by the cleanup job to keep the table bounded.
	DeletePastOccurrences(ctx context.Context, before time.Time) error

	// ListActiveAt returns all occurrences that are ACTIVE and whose window
	// contains `at`: occurrence_start <= at AND occurrence_end > at.
	ListActiveAt(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error)

	// ListActiveByPlacementsAt returns active occurrences for specific placement names.
	ListActiveByPlacementsAt(ctx context.Context, placementNames []string, at time.Time) ([]*entity.ScheduleOccurrence, error)

	// ListByScheduleID returns a paginated list of occurrences for a given
	// schedule, ordered by occurrence_start ascending.
	// Returns the matching rows and the total row count (for pagination).
	ListByScheduleID(ctx context.Context, scheduleID uuid.UUID, page, limit int) ([]*entity.ScheduleOccurrence, int64, error)
}
