package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// ScheduleOccurrenceService provides business logic for querying materialized schedule occurrences.
type ScheduleOccurrenceService struct {
	repo domainrepo.ScheduleOccurrenceRepository
}

// NewScheduleOccurrenceService creates a new ScheduleOccurrenceService.
func NewScheduleOccurrenceService(repo domainrepo.ScheduleOccurrenceRepository) *ScheduleOccurrenceService {
	return &ScheduleOccurrenceService{repo: repo}
}

// ListByScheduleID returns a paginated list of occurrences for a given schedule.
func (s *ScheduleOccurrenceService) ListByScheduleID(ctx context.Context, scheduleID uuid.UUID, page, limit int) ([]*entity.ScheduleOccurrence, int64, error) {
	return s.repo.ListByScheduleID(ctx, scheduleID, page, limit)
}

// ListActiveAt returns all occurrences that are active at the given time.
func (s *ScheduleOccurrenceService) ListActiveAt(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error) {
	return s.repo.ListActiveAt(ctx, at)
}
