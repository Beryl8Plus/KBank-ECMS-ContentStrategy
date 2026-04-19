package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// SchedulePostgresRepository implements domainrepo.ScheduleRepository using GORM.
type SchedulePostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.ScheduleRepository = (*SchedulePostgresRepository)(nil)

// NewSchedulePostgresRepository creates a new SchedulePostgresRepository.
func NewSchedulePostgresRepository(db *gorm.DB) *SchedulePostgresRepository {
	return &SchedulePostgresRepository{db: db}
}

// CheckScheduleOverlap returns the first active non-deleted schedule for the same
// (decision_rule_id, placement_id) pair whose [effective_from, effective_until) range
// overlaps [effectiveFrom, effectiveUntil).
//
// Overlap condition for half-open ranges [A.from, A.until) and [B.from, B.until):
//
//	A.from < B.until AND A.until > B.from
//
// GORM automatically adds deleted_at IS NULL via BaseModel's gorm.DeletedAt embed.
// Returns (nil, nil) when no conflicting schedule is found.
func (r *SchedulePostgresRepository) CheckScheduleOverlap(
	ctx context.Context,
	decisionRuleID uuid.UUID,
	placementID uuid.UUID,
	effectiveFrom, effectiveUntil time.Time,
	excludeID *uuid.UUID,
) (*entity.Schedule, error) {
	var conflict entity.Schedule

	q := r.db.WithContext(ctx).
		Where("\"DECISION_RULE_ID\" = ?", decisionRuleID).
		Where("\"PLACEMENT_ID\" = ?", placementID).
		Where("\"IS_ACTIVE\" = true").
		Where("\"EFFECTIVE_FROM\" < ? AND \"EFFECTIVE_UNTIL\" > ?", effectiveUntil, effectiveFrom)

	if excludeID != nil {
		q = q.Where("\"ID\" != ?", *excludeID)
	}

	err := q.First(&conflict).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("checking schedule overlap: %w", err)
	}

	return &conflict, nil
}

// CreateSchedule persists a new schedule record.
func (r *SchedulePostgresRepository) CreateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return fmt.Errorf("creating schedule: %w", err)
	}
	return nil
}

// GetScheduleByID retrieves a single non-deleted schedule by primary key.
// Returns (nil, nil) when no record is found.
func (r *SchedulePostgresRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	var s entity.Schedule
	err := r.db.WithContext(ctx).First(&s, "\"ID\" = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting schedule by ID: %w", err)
	}
	return &s, nil
}

// ListSchedules returns all non-deleted schedules ordered by created_at descending.
func (r *SchedulePostgresRepository) ListSchedules(ctx context.Context) ([]*entity.Schedule, error) {
	var schedules []*entity.Schedule
	if err := r.db.WithContext(ctx).Order("\"CREATED_AT\" DESC").Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("listing schedules: %w", err)
	}
	return schedules, nil
}

// ListSchedulesPaginated returns a page of non-deleted schedules ordered by created_at
// descending together with the total count. page and limit are 1-based and must be >= 1.
func (r *SchedulePostgresRepository) ListSchedulesPaginated(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error) {
	var schedules []*entity.Schedule
	var total int64

	base := r.db.WithContext(ctx).Model(&entity.Schedule{})
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting schedules: %w", err)
	}

	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).
		Order("\"CREATED_AT\" DESC").
		Limit(limit).
		Offset(offset).
		Find(&schedules).Error; err != nil {
		return nil, 0, fmt.Errorf("listing schedules paginated: %w", err)
	}

	return schedules, total, nil
}

// UpdateSchedule saves all fields of an existing schedule record.
func (r *SchedulePostgresRepository) UpdateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return fmt.Errorf("updating schedule: %w", err)
	}
	return nil
}

// DeleteSchedule soft-deletes the schedule with the given ID.
func (r *SchedulePostgresRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.Schedule{}, "\"ID\" = ?", id).Error; err != nil {
		return fmt.Errorf("deleting schedule: %w", err)
	}
	return nil
}

// ListActiveSchedulesInWindow returns all active schedules whose time window
// contains at: effective_from <= at AND effective_until > at.
// Each schedule is preloaded with its DecisionRule and Placement associations.
func (r *SchedulePostgresRepository) ListActiveSchedulesInWindow(ctx context.Context, at time.Time) ([]*entity.Schedule, error) {
	var atStr = at.Format(time.RFC3339)
	var schedules []*entity.Schedule
	err := r.db.WithContext(ctx).
		Preload("DecisionRule", func(db *gorm.DB) *gorm.DB {
			return db.Where("\"STATUS\" = ?", enums.DecisionRuleStatusActive)
		}).
		Preload("DecisionRule.RuleConditions").
		Preload("DecisionRule.RuleConditions.Attribute").
		Preload("DecisionRule.Rules.RuleAttributes").
		Preload("Placement").
		Where("\"IS_ACTIVE\" = true").
		Where("\"EFFECTIVE_FROM\" <= ? AND \"EFFECTIVE_UNTIL\" > ?", atStr, atStr).
		Find(&schedules).Error
	if err != nil {
		return nil, fmt.Errorf("listing active schedules in window: %w", err)
	}
	return schedules, nil
}
