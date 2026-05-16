package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// ScheduleOccurrencePostgresRepository implements
// domainrepo.ScheduleOccurrenceRepository using GORM.
type ScheduleOccurrencePostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.ScheduleOccurrenceRepository = (*ScheduleOccurrencePostgresRepository)(nil)

// NewScheduleOccurrencePostgresRepository creates a new
// ScheduleOccurrencePostgresRepository.
func NewScheduleOccurrencePostgresRepository(db *gorm.DB) *ScheduleOccurrencePostgresRepository {
	return &ScheduleOccurrencePostgresRepository{db: db}
}

// UpsertOccurrences inserts or updates schedule occurrences in bulk.
// The ON CONFLICT clause targets the unique index on
// (schedule_id, occurrence_start, occurrence_end) and updates the status,
// source, and updated_at timestamp on conflict so that the operation is
// idempotent — re-running the materialisation job for the same window
// produces no duplicate rows.
func (r *ScheduleOccurrencePostgresRepository) UpsertOccurrences(
	ctx context.Context,
	occurrences []*entity.ScheduleOccurrence,
) error {
	if len(occurrences) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "SCHEDULE_ID"},
				{Name: "OCCURRENCE_START"},
				{Name: "OCCURRENCE_END"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"STATUS",
				"SOURCE",
				"UPDATED_AT",
				"UPDATED_BY",
			}),
		}).
		Create(&occurrences).Error; err != nil {
		return fmt.Errorf("upserting schedule occurrences: %w", err)
	}
	return nil
}

// DeleteFutureByScheduleID removes all occurrences for the given schedule
// whose occurrence_start is strictly after `after`.
// This is called before re-materialising when a Schedule has been updated
// or deleted so that no stale future windows remain.
func (r *ScheduleOccurrencePostgresRepository) DeleteFutureByScheduleID(
	ctx context.Context,
	scheduleID uuid.UUID,
	after time.Time,
) error {
	if err := r.db.WithContext(ctx).
		Unscoped(). // hard-delete: occurrences don't need soft-delete semantics
		Where(`"SCHEDULE_ID" = ? AND "OCCURRENCE_START" > ?`, scheduleID, after).
		Delete(&entity.ScheduleOccurrence{}).Error; err != nil {
		return fmt.Errorf("deleting future occurrences for schedule %s: %w", scheduleID, err)
	}
	return nil
}

// DeletePastOccurrences removes all occurrences whose occurrence_end is
// strictly before `before`. This is the cleanup job entry point and
// performs a hard delete to reclaim storage.
func (r *ScheduleOccurrencePostgresRepository) DeletePastOccurrences(
	ctx context.Context,
	before time.Time,
) error {
	if err := r.db.WithContext(ctx).
		Unscoped(). // hard-delete for cleanup
		Where(`"OCCURRENCE_END" < ?`, before).
		Delete(&entity.ScheduleOccurrence{}).Error; err != nil {
		return fmt.Errorf("deleting past occurrences before %s: %w", before.Format(time.RFC3339), err)
	}
	return nil
}

// ListByScheduleID returns a paginated list of occurrences for a given schedule,
// ordered by occurrence_start ascending.
func (r *ScheduleOccurrencePostgresRepository) ListByScheduleID(
	ctx context.Context,
	scheduleID uuid.UUID,
	page, limit int,
) ([]*entity.ScheduleOccurrence, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	var occurrences []*entity.ScheduleOccurrence
	var total int64

	base := r.db.WithContext(ctx).Model(&entity.ScheduleOccurrence{}).
		Where(`"SCHEDULE_ID" = ?`, scheduleID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting occurrences for schedule %s: %w", scheduleID, err)
	}

	offset := (page - 1) * limit
	if err := base.Order(`"OCCURRENCE_START" ASC`).
		Offset(offset).Limit(limit).
		Find(&occurrences).Error; err != nil {
		return nil, 0, fmt.Errorf("listing occurrences for schedule %s: %w", scheduleID, err)
	}

	return occurrences, total, nil
}

// ListActiveAt returns all ACTIVE occurrences whose parent decision rule is
// ACTIVE and whose window contains `at`:
//
//	occurrence_start <= at AND occurrence_end > at
//
// Each occurrence is preloaded with its parent Schedule.
func (r *ScheduleOccurrencePostgresRepository) ListActiveAt(
	ctx context.Context,
	at time.Time,
) ([]*entity.ScheduleOccurrence, error) {
	var occurrences []*entity.ScheduleOccurrence
	if err := r.db.WithContext(ctx).
		Preload(`Schedule`).
		Preload(`Schedule.Placement`).
		Joins(`JOIN "schedules" ON "schedules"."ID" = "schedule_occurrences"."SCHEDULE_ID"`).
		Joins(`JOIN "decision_rules" ON "decision_rules"."ID" = "schedules"."DECISION_RULE_ID"`).
		Where(`"schedule_occurrences"."STATUS" = ?`, enums.OccurrenceStatusActive).
		Where(`"schedule_occurrences"."OCCURRENCE_START" <= ? AND "schedule_occurrences"."OCCURRENCE_END" > ?`, at, at).
		Where(`"decision_rules"."STATUS" = ?`, enums.DecisionRuleStatusActive).
		Find(&occurrences).Error; err != nil {
		return nil, fmt.Errorf("listing active occurrences at %s: %w", at.Format(time.RFC3339), err)
	}
	return occurrences, nil
}

// ExpireEndedOccurrences bulk-updates every ACTIVE occurrence whose
// occurrence_end ≤ now to EXPIRED in a single UPDATE statement.
// Returns the number of rows affected.
func (r *ScheduleOccurrencePostgresRepository) ExpireEndedOccurrences(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&entity.ScheduleOccurrence{}).
		Where(`"STATUS" = ? AND "OCCURRENCE_END" <= ?`, enums.OccurrenceStatusActive, now).
		Update("STATUS", enums.OccurrenceStatusExpired)
	if result.Error != nil {
		return 0, fmt.Errorf("expiring ended occurrences: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// ListActiveByPlacementsAt returns active occurrences whose parent decision
// rule is ACTIVE for specific placement names.
func (r *ScheduleOccurrencePostgresRepository) ListActiveByPlacementsAt(
	ctx context.Context,
	placementNames []string,
	at time.Time,
) ([]*entity.ScheduleOccurrence, error) {
	if len(placementNames) == 0 {
		return nil, nil
	}

	var occurrences []*entity.ScheduleOccurrence
	if err := r.db.WithContext(ctx).
		Joins(`JOIN "schedules" ON "schedules"."ID" = "schedule_occurrences"."SCHEDULE_ID"`).
		Joins(`JOIN "placements" ON "placements"."ID" = "schedules"."PLACEMENT_ID"`).
		Joins(`JOIN "decision_rules" ON "decision_rules"."ID" = "schedules"."DECISION_RULE_ID"`).
		Preload(`Schedule`).
		Preload(`Schedule.Placement`).
		Where(`"placements"."PLACEMENT_NAME" IN ?`, placementNames).
		Where(`"schedule_occurrences"."STATUS" = ?`, enums.OccurrenceStatusActive).
		Where(`"schedule_occurrences"."OCCURRENCE_START" <= ? AND "schedule_occurrences"."OCCURRENCE_END" > ?`, at, at).
		Where(`"decision_rules"."STATUS" = ?`, enums.DecisionRuleStatusActive).
		Find(&occurrences).Error; err != nil {
		return nil, fmt.Errorf("listing active occurrences for placements %v at %s: %w", placementNames, at.Format(time.RFC3339), err)
	}
	return occurrences, nil
}

// CancelByDecisionRuleID bulk-updates every ACTIVE occurrence whose parent
// Schedule belongs to the given decision rule to CANCELLED. Uses a subquery
// to scope by SCHEDULE_ID so the operation completes in a single round-trip
// regardless of how many schedules the decision rule owns.
// Returns the number of rows affected.
func (r *ScheduleOccurrencePostgresRepository) CancelByDecisionRuleID(
	ctx context.Context,
	decisionRuleID uuid.UUID,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&entity.ScheduleOccurrence{}).
		Where(
			`"STATUS" = ? AND "SCHEDULE_ID" IN (?)`,
			enums.OccurrenceStatusActive,
			r.db.WithContext(ctx).Model(&entity.Schedule{}).
				Select(`"ID"`).
				Where(`"DECISION_RULE_ID" = ?`, decisionRuleID),
		).
		Update("STATUS", enums.OccurrenceStatusCancelled)
	if result.Error != nil {
		return 0, fmt.Errorf("cancelling occurrences for decision rule %s: %w", decisionRuleID, result.Error)
	}
	return result.RowsAffected, nil
}
