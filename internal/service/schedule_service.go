package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// ScheduleService encapsulates business logic for schedule management.
type ScheduleService struct {
	repo domainrepo.ScheduleRepository
}

// NewScheduleService creates a new ScheduleService.
func NewScheduleService(repo domainrepo.ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

// ValidateScheduleOverlap checks whether the given schedule would overlap with any
// existing active schedule for the same placement. Returns a descriptive error if an
// overlap is found, or nil if the schedule may proceed.
//
// When schedule.ID is non-zero (existing record), it is excluded from the check so
// that updating a schedule does not falsely conflict with itself.
func (s *ScheduleService) ValidateScheduleOverlap(ctx context.Context, schedule *entity.Schedule) error {
	var excludeID *uuid.UUID
	if schedule.ID != (uuid.UUID{}) {
		excludeID = &schedule.ID
	}

	conflict, err := s.repo.CheckScheduleOverlap(
		ctx,
		schedule.PlacementID,
		schedule.EffectiveFrom,
		schedule.EffectiveUntil,
		excludeID,
	)
	if err != nil {
		return fmt.Errorf("validating schedule overlap: %w", err)
	}

	if conflict == nil {
		return nil
	}

	return fmt.Errorf(
		"schedule overlaps with existing schedule %s for placement %s from %s to %s",
		conflict.ID,
		conflict.PlacementID,
		conflict.EffectiveFrom.Format("2006-01-02"),
		conflict.EffectiveUntil.Format("2006-01-02"),
	)
}

// CreateSchedule validates overlap and persists a new schedule.
func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if err := s.ValidateScheduleOverlap(ctx, schedule); err != nil {
		return err
	}
	return s.repo.CreateSchedule(ctx, schedule)
}

// GetScheduleByID retrieves a single schedule by its ID.
// Returns (nil, nil) when not found.
func (s *ScheduleService) GetScheduleByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	return s.repo.GetScheduleByID(ctx, id)
}

// ListSchedules returns all non-deleted schedules.
func (s *ScheduleService) ListSchedules(ctx context.Context) ([]*entity.Schedule, error) {
	return s.repo.ListSchedules(ctx)
}

// ListSchedulesPaginated returns a page of non-deleted schedules and the total record count.
// page and limit must be >= 1.
func (s *ScheduleService) ListSchedulesPaginated(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error) {
	return s.repo.ListSchedulesPaginated(ctx, page, limit)
}

// UpdateSchedule validates overlap (excluding itself) and saves the updated schedule.
func (s *ScheduleService) UpdateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if err := s.ValidateScheduleOverlap(ctx, schedule); err != nil {
		return err
	}
	return s.repo.UpdateSchedule(ctx, schedule)
}

// DeleteSchedule soft-deletes a schedule by ID.
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSchedule(ctx, id)
}
