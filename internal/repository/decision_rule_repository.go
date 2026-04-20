package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// DecisionRulePostgresRepository implements domainrepo.DecisionRuleRepository using GORM.
type DecisionRulePostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.DecisionRuleRepository = (*DecisionRulePostgresRepository)(nil)

// NewDecisionRulePostgresRepository creates a new DecisionRulePostgresRepository.
func NewDecisionRulePostgresRepository(db *gorm.DB) *DecisionRulePostgresRepository {
	return &DecisionRulePostgresRepository{db: db}
}

// GetDecisionRuleByScheduleID retrieves the DecisionRule associated with the given
// schedule ID, preloaded with RuleConditions, Attributes, Rules, and RuleAttributes.
// Returns (nil, nil) when no matching schedule or decision rule is found.
func (r *DecisionRulePostgresRepository) GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error) {
	var schedule entity.Schedule

	// First find the schedule to get the decision rule ID.
	err := r.db.WithContext(ctx).
		Select("\"DECISION_RULE_ID\"").
		Where("\"ID\" = ?", scheduleID).
		First(&schedule).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting schedule to find decision rule: %w", err)
	}

	var decisionRule entity.DecisionRule
	err = r.db.WithContext(ctx).
		Preload("RuleConditions").
		Preload("RuleConditions.Attribute").
		Preload("Rules").
		Preload("Rules.RuleAttributes").
		Where("\"ID\" = ?", schedule.DecisionRuleID).
		First(&decisionRule).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting decision rule by schedule id: %w", err)
	}

	return &decisionRule, nil
}
