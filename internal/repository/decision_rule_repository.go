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

// GetDecisionRuleByScheduleIDs retrieves the DecisionRules associated with the given
// schedule IDs, preloaded with RuleConditions, Attributes, Rules, and RuleAttributes.
// Returns a map of scheduleID to DecisionRule. ScheduleIDs with no matching decision rule are omitted.
func (r *DecisionRulePostgresRepository) GetDecisionRuleByScheduleIDs(ctx context.Context, scheduleIDs []uuid.UUID) (map[uuid.UUID]*entity.DecisionRule, error) {
	var schedules []entity.Schedule
	err := r.db.WithContext(ctx).
		Select(`"ID"`, `"DECISION_RULE_ID"`).
		Where(`"ID" IN ?`, scheduleIDs).
		Find(&schedules).Error
	if err != nil {
		return nil, fmt.Errorf("getting schedules by ids: %w", err)
	}

	drIDToScheduleIDs := make(map[uuid.UUID][]uuid.UUID)
	for _, s := range schedules {
		if s.DecisionRuleID != uuid.Nil {
			drIDToScheduleIDs[s.DecisionRuleID] = append(drIDToScheduleIDs[s.DecisionRuleID], s.ID)
		}
	}

	if len(drIDToScheduleIDs) == 0 {
		return make(map[uuid.UUID]*entity.DecisionRule), nil
	}

	drIDs := make([]uuid.UUID, 0, len(drIDToScheduleIDs))
	for id := range drIDToScheduleIDs {
		drIDs = append(drIDs, id)
	}

	var decisionRules []entity.DecisionRule
	err = r.db.WithContext(ctx).
		Where(`"ID" IN ?`, drIDs).
		Preload("RuleConditions").
		Preload("RuleConditions.Attribute").
		Preload("Rules").
		Preload("Rules.RuleAttributes").
		Find(&decisionRules).Error
	if err != nil {
		return nil, fmt.Errorf("getting decision rules by ids: %w", err)
	}

	drByID := make(map[uuid.UUID]*entity.DecisionRule, len(decisionRules))
	for i := range decisionRules {
		drByID[decisionRules[i].ID] = &decisionRules[i]
	}

	result := make(map[uuid.UUID]*entity.DecisionRule)
	for drID, sIDs := range drIDToScheduleIDs {
		dr, ok := drByID[drID]
		if !ok {
			continue
		}
		for _, sID := range sIDs {
			result[sID] = dr
		}
	}
	return result, nil
}
