package service

import (
	"context"
	"fmt"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"

	"github.com/google/uuid"
)

// IntegrityCheckerService inspects all non-INACTIVE DecisionRules and marks
// as INACTIVE any rule that references a deactivated or value-mismatched attribute.
type IntegrityCheckerService struct {
	repo domainrepo.IntegrityRepository
}

// NewIntegrityCheckerService creates a new IntegrityCheckerService.
func NewIntegrityCheckerService(repo domainrepo.IntegrityRepository) *IntegrityCheckerService {
	return &IntegrityCheckerService{repo: repo}
}

// RunCheck performs the full integrity scan and marks violating rules inactive.
// It uses a Postgres advisory lock to prevent concurrent runs across instances.
func (s *IntegrityCheckerService) RunCheck(ctx context.Context) error {
	acquired, err := s.repo.TryAcquireCheckerLock(ctx)
	if err != nil {
		return fmt.Errorf("acquire checker lock: %w", err)
	}
	if !acquired {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "INTEGRITY-CHECKER",
			Level:   "INFO",
			Message: "skipping run: another instance holds the lock",
		})
		return nil
	}
	defer func() {
		if releaseErr := s.repo.ReleaseCheckerLock(ctx); releaseErr != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "INTEGRITY-CHECKER",
				Level:   "ERROR",
				Message: "release lock failed: " + releaseErr.Error(),
			})
		}
	}()

	var violatingIDs []uuid.UUID

	inactiveDRs, err := s.repo.FindDecisionRulesWithInactiveAttributes(ctx)
	if err != nil {
		return fmt.Errorf("find DRs with inactive attrs: %w", err)
	}
	violatingIDs = append(violatingIDs, inactiveDRs...)

	invalidValueDRs, err := s.repo.FindDecisionRulesWithInvalidValues(ctx)
	if err != nil {
		return fmt.Errorf("find DRs with invalid values: %w", err)
	}
	violatingIDs = append(violatingIDs, invalidValueDRs...)

	unique := deduplicateUUIDs(violatingIDs)
	if len(unique) == 0 {
		return nil
	}

	if err := s.repo.MarkDecisionRulesInactive(ctx, unique); err != nil {
		return fmt.Errorf("mark inactive: %w", err)
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "INTEGRITY-CHECKER",
		Level:   "INFO",
		Message: fmt.Sprintf("marked %d decision rule(s) inactive", len(unique)),
	})
	return nil
}

func deduplicateUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := ids[:0]
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
