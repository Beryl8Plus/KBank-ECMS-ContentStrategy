package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// DecisionRuleWizardPostgresRepository implements domainrepo.DecisionRuleWizardRepository.
type DecisionRuleWizardPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.DecisionRuleWizardRepository = (*DecisionRuleWizardPostgresRepository)(nil)

// NewDecisionRuleWizardPostgresRepository creates a new DecisionRuleWizardPostgresRepository.
func NewDecisionRuleWizardPostgresRepository(db *gorm.DB) *DecisionRuleWizardPostgresRepository {
	return &DecisionRuleWizardPostgresRepository{db: db}
}

// SaveStep1 atomically creates a DecisionRule and all its template RuleConditions.
func (r *DecisionRuleWizardPostgresRepository) SaveStep1(ctx context.Context, dr *entity.DecisionRule, conditions []*entity.RuleCondition) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dr).Error; err != nil {
			return fmt.Errorf("creating decision rule: %w", err)
		}
		if len(conditions) == 0 {
			return nil
		}
		if err := tx.CreateInBatches(conditions, 100).Error; err != nil {
			return fmt.Errorf("creating rule conditions: %w", err)
		}
		return nil
	})
}

// FindDecisionRuleByID returns a DecisionRule by primary key, or (nil, nil) when not found.
func (r *DecisionRuleWizardPostgresRepository) FindDecisionRuleByID(ctx context.Context, id uuid.UUID) (*entity.DecisionRule, error) {
	var dr entity.DecisionRule
	err := r.db.WithContext(ctx).First(&dr, `"ID" = ?`, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding decision rule by id: %w", err)
	}
	return &dr, nil
}

// FindTemplateConditions returns all template RuleConditions (no rule association)
// for the given DecisionRule, with Attribute preloaded.
func (r *DecisionRuleWizardPostgresRepository) FindTemplateConditions(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.RuleCondition, error) {
	var conditions []*entity.RuleCondition
	err := r.db.WithContext(ctx).
		Joins("Attribute").
		Where(`rule_conditions."DECISION_RULE_ID" = ?`, decisionRuleID).
		Order(`rule_conditions."PARENT_RULE_CONDITION_ID" NULLS FIRST, rule_conditions."SEQUENCE"`).
		Find(&conditions).Error
	if err != nil {
		return nil, fmt.Errorf("finding template conditions: %w", err)
	}
	return conditions, nil
}

// FindConditionsByIDs returns RuleConditions matching the given IDs.
func (r *DecisionRuleWizardPostgresRepository) FindConditionsByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.RuleCondition, error) {
	var conditions []*entity.RuleCondition
	err := r.db.WithContext(ctx).
		Where(`"ID" IN ?`, ids).
		Find(&conditions).Error
	if err != nil {
		return nil, fmt.Errorf("finding conditions by ids: %w", err)
	}
	return conditions, nil
}

// FindRulesByDecisionRuleID returns all Rules for a DecisionRule ordered by ORDER_NO.
func (r *DecisionRuleWizardPostgresRepository) FindRulesByDecisionRuleID(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.Rule, error) {
	var rules []*entity.Rule
	err := r.db.WithContext(ctx).
		Where(`"DECISION_RULE_ID" = ?`, decisionRuleID).
		Order(`"ORDER_NO"`).
		Find(&rules).Error
	if err != nil {
		return nil, fmt.Errorf("finding rules by decision rule id: %w", err)
	}
	return rules, nil
}

// FindRuleAttributesByRuleIDs returns all RuleAttributes for the given rule IDs
// with Attribute preloaded.
func (r *DecisionRuleWizardPostgresRepository) FindRuleAttributesByRuleIDs(ctx context.Context, ruleIDs []uuid.UUID) ([]*entity.RuleAttribute, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	var attrs []*entity.RuleAttribute
	err := r.db.WithContext(ctx).
		Joins("Attribute").
		Where(`rule_attributes."RULE_ID" IN ?`, ruleIDs).
		Find(&attrs).Error
	if err != nil {
		return nil, fmt.Errorf("finding rule attributes by rule ids: %w", err)
	}
	return attrs, nil
}

// FindRuleByID returns a Rule by primary key, or (nil, nil) when not found.
func (r *DecisionRuleWizardPostgresRepository) FindRuleByID(ctx context.Context, id uuid.UUID) (*entity.Rule, error) {
	var rule entity.Rule
	err := r.db.WithContext(ctx).First(&rule, `"ID" = ?`, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding rule by id: %w", err)
	}
	return &rule, nil
}

// SaveStep2 atomically upserts Rules and replaces their RuleAttributes.
func (r *DecisionRuleWizardPostgresRepository) SaveStep2(ctx context.Context, rules []*entity.Rule, attrs []*entity.RuleAttribute) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			if rule.ID == uuid.Nil {
				if err := tx.Create(rule).Error; err != nil {
					return fmt.Errorf("creating rule: %w", err)
				}
			} else {
				if err := tx.Save(rule).Error; err != nil {
					return fmt.Errorf("updating rule: %w", err)
				}
			}
		}

		// Group attrs by rule ID, delete existing, then insert new.
		attrsByRule := make(map[uuid.UUID][]*entity.RuleAttribute)
		for _, a := range attrs {
			attrsByRule[a.RuleID] = append(attrsByRule[a.RuleID], a)
		}
		for ruleID, ruleAttrs := range attrsByRule {
			if err := tx.Where(`"RULE_ID" = ?`, ruleID).Delete(&entity.RuleAttribute{}).Error; err != nil {
				return fmt.Errorf("deleting existing rule attributes for rule %s: %w", ruleID, err)
			}
			if len(ruleAttrs) > 0 {
				if err := tx.CreateInBatches(ruleAttrs, 100).Error; err != nil {
					return fmt.Errorf("creating rule attributes: %w", err)
				}
			}
		}
		return nil
	})
}

// CountSchedulesByPlacement counts non-deleted schedules for a placement.
func (r *DecisionRuleWizardPostgresRepository) CountSchedulesByPlacement(ctx context.Context, placementID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Schedule{}).
		Where(`"PLACEMENT_ID" = ?`, placementID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting schedules by placement: %w", err)
	}
	return count, nil
}

// ExistsSchedule reports whether a schedule already exists for (decisionRuleID, placementID).
func (r *DecisionRuleWizardPostgresRepository) ExistsSchedule(ctx context.Context, decisionRuleID, placementID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Schedule{}).
		Where(`"DECISION_RULE_ID" = ? AND "PLACEMENT_ID" = ?`, decisionRuleID, placementID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking schedule exists: %w", err)
	}
	return count > 0, nil
}

// SaveStep3 atomically inserts schedules for a DecisionRule.
func (r *DecisionRuleWizardPostgresRepository) SaveStep3(ctx context.Context, decisionRuleID uuid.UUID, schedules []*entity.Schedule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(schedules, 50).Error; err != nil {
			return fmt.Errorf("creating schedules: %w", err)
		}
		return nil
	})
}

// ActivateDecisionRule sets the DecisionRule status to ACTIVE.
func (r *DecisionRuleWizardPostgresRepository) ActivateDecisionRule(ctx context.Context, decisionRuleID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&entity.DecisionRule{}).
		Where(`"ID" = ?`, decisionRuleID).
		Update(`"STATUS"`, enums.DecisionRuleStatusActive).Error
	if err != nil {
		return fmt.Errorf("activating decision rule: %w", err)
	}
	return nil
}

// ListDecisionRules returns a paginated list of DecisionRules matching the filter.
func (r *DecisionRuleWizardPostgresRepository) ListDecisionRules(ctx context.Context, f domainrepo.DecisionRuleListFilter) ([]*entity.DecisionRule, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.DecisionRule{})

	if f.Type != "" {
		q = q.Where(`"TYPE" = ?`, f.Type)
	}
	if f.EvaluateType != "" {
		q = q.Where(`"EVALUATE_TYPE" = ?`, f.EvaluateType)
	}
	if f.Status != "" {
		q = q.Where(`"STATUS" = ?`, f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where(`"NAME" ILIKE ? OR "DECISION_RULE_ID" ILIKE ?`, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting decision rules: %w", err)
	}

	var drs []*entity.DecisionRule
	q = q.Order(`"CREATED_AT" DESC`)
	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset((f.Page - 1) * f.Limit)
	}
	err := q.Find(&drs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing decision rules: %w", err)
	}
	return drs, total, nil
}

// FindSchedulesByDecisionRuleID returns Schedules with Placement and Channel preloaded
// for a single decision rule.
func (r *DecisionRuleWizardPostgresRepository) FindSchedulesByDecisionRuleID(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.Schedule, error) {
	var schedules []*entity.Schedule
	err := r.db.WithContext(ctx).
		Joins("Placement").
		Joins("Placement.Channel").
		Where(`schedules."DECISION_RULE_ID" = ?`, decisionRuleID).
		Find(&schedules).Error
	if err != nil {
		return nil, fmt.Errorf("finding schedules by decision rule id: %w", err)
	}
	return schedules, nil
}

// FindSchedulesWithPlacementsByDecisionRuleIDs returns Schedules with Placement and Channel
// preloaded for the given decision rule IDs.
func (r *DecisionRuleWizardPostgresRepository) FindSchedulesWithPlacementsByDecisionRuleIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Schedule, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var schedules []*entity.Schedule
	err := r.db.WithContext(ctx).
		Joins("Placement").
		Joins("Placement.Channel").
		Where(`schedules."DECISION_RULE_ID" IN ?`, ids).
		Find(&schedules).Error
	if err != nil {
		return nil, fmt.Errorf("finding schedules with placements: %w", err)
	}
	return schedules, nil
}
