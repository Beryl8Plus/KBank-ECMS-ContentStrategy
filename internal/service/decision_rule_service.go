package service

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// DecisionRuleService encapsulates business logic for decision rule management.
type DecisionRuleService struct {
	repo domainrepo.DecisionRuleRepository
}

// NewDecisionRuleService creates a new DecisionRuleService.
func NewDecisionRuleService(repo domainrepo.DecisionRuleRepository) *DecisionRuleService {
	return &DecisionRuleService{repo: repo}
}

// GetDecisionRuleByScheduleID retrieves a decision rule by its associated schedule ID.
// Returns (nil, nil) when not found.
func (s *DecisionRuleService) GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error) {
	return s.repo.GetDecisionRuleByScheduleID(ctx, scheduleID)
}
