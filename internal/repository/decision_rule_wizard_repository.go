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

// SaveStep2 atomically deletes removed rules, upserts remaining rules, and replaces their RuleAttributes.
func (r *DecisionRuleWizardPostgresRepository) SaveStep2(ctx context.Context, rules []*entity.Rule, attrs []*entity.RuleAttribute, toDeleteRuleIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete rules removed from the request (with their attributes).
		if len(toDeleteRuleIDs) > 0 {
			if err := tx.Where(`"RULE_ID" IN ?`, toDeleteRuleIDs).Delete(&entity.RuleAttribute{}).Error; err != nil {
				return fmt.Errorf("deleting rule attributes for removed rules: %w", err)
			}
			if err := tx.Where(`"ID" IN ?`, toDeleteRuleIDs).Delete(&entity.Rule{}).Error; err != nil {
				return fmt.Errorf("deleting removed rules: %w", err)
			}
		}

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

// CountSchedulesByPlacementExcludingDR counts non-deleted schedules for a placement
// that belong to other decision rules (excludes excludeDRID).
func (r *DecisionRuleWizardPostgresRepository) CountSchedulesByPlacementExcludingDR(ctx context.Context, placementID, excludeDRID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Schedule{}).
		Where(`"PLACEMENT_ID" = ? AND "DECISION_RULE_ID" != ?`, placementID, excludeDRID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting schedules by placement excluding dr: %w", err)
	}
	return count, nil
}

// SaveStep3 atomically replaces all schedules for a DecisionRule:
// deletes every existing schedule for the DR, then inserts the new set.
func (r *DecisionRuleWizardPostgresRepository) SaveStep3(ctx context.Context, decisionRuleID uuid.UUID, schedules []*entity.Schedule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(`"DECISION_RULE_ID" = ?`, decisionRuleID).Delete(&entity.Schedule{}).Error; err != nil {
			return fmt.Errorf("deleting existing schedules: %w", err)
		}
		if len(schedules) == 0 {
			return nil
		}
		if err := tx.CreateInBatches(schedules, 50).Error; err != nil {
			return fmt.Errorf("creating schedules: %w", err)
		}
		return nil
	})
}

// ActivateDecisionRule sets STATUS = ACTIVE and resets SUB_STATUS = "N/A".
func (r *DecisionRuleWizardPostgresRepository) ActivateDecisionRule(ctx context.Context, decisionRuleID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&entity.DecisionRule{}).
		Where(`"ID" = ?`, decisionRuleID).
		Updates(map[string]any{
			"STATUS":     enums.DecisionRuleStatusActive,
			"SUB_STATUS": enums.DecisionRuleSubStatusNA,
		}).Error
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
	err := q.Preload("CreatedByUser").Preload("UpdatedByUser").Preload("InactiveByUser").Find(&drs).Error
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

// UpdateStep1 atomically updates a DecisionRule header, upserts conditions, and cascades
// deletes into Step 2 for any removed conditions. Returns IDs of deleted Rules.
func (r *DecisionRuleWizardPostgresRepository) UpdateStep1(
	ctx context.Context,
	id uuid.UUID,
	dr *entity.DecisionRule,
	toUpsert []*entity.RuleCondition,
	toDeleteConditionIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	var affectedRuleIDs []uuid.UUID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ── Cascade delete Step 2 data for removed conditions ──────────────────
		if len(toDeleteConditionIDs) > 0 {
			// Collect attributeIDs of the conditions being deleted (leaf nodes only).
			var toDeleteConds []*entity.RuleCondition
			if err := tx.Where(`"ID" IN ?`, toDeleteConditionIDs).Find(&toDeleteConds).Error; err != nil {
				return fmt.Errorf("fetching conditions to delete: %w", err)
			}
			deletedAttrIDs := make([]uuid.UUID, 0, len(toDeleteConds))
			for _, c := range toDeleteConds {
				if c.AttributeID != uuid.Nil {
					deletedAttrIDs = append(deletedAttrIDs, c.AttributeID)
				}
			}

			if len(deletedAttrIDs) > 0 {
				// Find Rules that have at least one RuleAttribute for a deleted attribute.
				type ruleIDRow struct{ ID uuid.UUID }
				var rows []ruleIDRow
				if err := tx.Raw(`
					SELECT DISTINCT r."ID"
					FROM rules r
					INNER JOIN rule_attributes ra ON ra."RULE_ID" = r."ID"
					WHERE r."DECISION_RULE_ID" = ?
					  AND ra."ATTRIBUTE_ID" IN ?
					  AND r."DELETED_AT" IS NULL
					  AND ra."DELETED_AT" IS NULL
				`, id, deletedAttrIDs).Scan(&rows).Error; err != nil {
					return fmt.Errorf("finding affected rules: %w", err)
				}
				for _, row := range rows {
					affectedRuleIDs = append(affectedRuleIDs, row.ID)
				}

				if len(affectedRuleIDs) > 0 {
					if err := tx.Where(`"RULE_ID" IN ?`, affectedRuleIDs).Delete(&entity.RuleAttribute{}).Error; err != nil {
						return fmt.Errorf("deleting rule attributes: %w", err)
					}
					if err := tx.Where(`"ID" IN ?`, affectedRuleIDs).Delete(&entity.Rule{}).Error; err != nil {
						return fmt.Errorf("deleting affected rules: %w", err)
					}
				}
			}

			// Collect all descendant condition IDs (iterative BFS, max depth = 3).
			allToDelete := make(map[uuid.UUID]struct{}, len(toDeleteConditionIDs))
			frontier := make([]uuid.UUID, len(toDeleteConditionIDs))
			for i, cid := range toDeleteConditionIDs {
				allToDelete[cid] = struct{}{}
				frontier[i] = cid
			}
			for len(frontier) > 0 {
				var children []*entity.RuleCondition
				if err := tx.Where(`"PARENT_RULE_CONDITION_ID" IN ?`, frontier).Find(&children).Error; err != nil {
					return fmt.Errorf("finding child conditions: %w", err)
				}
				frontier = frontier[:0]
				for _, child := range children {
					if _, seen := allToDelete[child.ID]; !seen {
						allToDelete[child.ID] = struct{}{}
						frontier = append(frontier, child.ID)
					}
				}
			}
			deleteIDs := make([]uuid.UUID, 0, len(allToDelete))
			for cid := range allToDelete {
				deleteIDs = append(deleteIDs, cid)
			}
			if err := tx.Where(`"ID" IN ?`, deleteIDs).Delete(&entity.RuleCondition{}).Error; err != nil {
				return fmt.Errorf("deleting conditions: %w", err)
			}
		}

		// ── Upsert conditions (parents are always before children in toUpsert) ──
		for _, cond := range toUpsert {
			if cond.ID == uuid.Nil {
				cond.ID = uuid.New()
				if err := tx.Create(cond).Error; err != nil {
					return fmt.Errorf("creating condition: %w", err)
				}
			} else {
				if err := tx.Save(cond).Error; err != nil {
					return fmt.Errorf("updating condition: %w", err)
				}
			}
		}

		// ── Update DecisionRule header fields ───────────────────────────────────
		updates := map[string]interface{}{
			`"NAME"`:          dr.Name,
			`"TYPE"`:          dr.Type,
			`"EVALUATE_TYPE"`: dr.EvaluateType,
			`"CONTENT_PATH"`:  dr.ContentPath,
			`"CAMPAIGN_CODE"`: dr.CampaignCode,
			`"SCORE"`:         dr.Score,
		}
		if err := tx.Model(&entity.DecisionRule{}).Where(`"ID" = ?`, id).Updates(updates).Error; err != nil {
			return fmt.Errorf("updating decision rule header: %w", err)
		}
		return nil
	})
	return affectedRuleIDs, err
}

// CloneDecisionRule atomically inserts a cloned DecisionRule with all child records.
func (r *DecisionRuleWizardPostgresRepository) CloneDecisionRule(
	ctx context.Context,
	dr *entity.DecisionRule,
	conditions []*entity.RuleCondition,
	rules []*entity.Rule,
	attrs []*entity.RuleAttribute,
	schedules []*entity.Schedule,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dr).Error; err != nil {
			return fmt.Errorf("creating cloned decision rule: %w", err)
		}
		if len(conditions) > 0 {
			if err := tx.CreateInBatches(conditions, 100).Error; err != nil {
				return fmt.Errorf("creating cloned conditions: %w", err)
			}
		}
		if len(rules) > 0 {
			if err := tx.CreateInBatches(rules, 100).Error; err != nil {
				return fmt.Errorf("creating cloned rules: %w", err)
			}
		}
		if len(attrs) > 0 {
			if err := tx.CreateInBatches(attrs, 100).Error; err != nil {
				return fmt.Errorf("creating cloned rule attributes: %w", err)
			}
		}
		if len(schedules) > 0 {
			if err := tx.CreateInBatches(schedules, 50).Error; err != nil {
				return fmt.Errorf("creating cloned schedules: %w", err)
			}
		}
		return nil
	})
}

// DeactivateDecisionRule sets STATUS=INACTIVE and records the user who deactivated.
func (r *DecisionRuleWizardPostgresRepository) DeactivateDecisionRule(ctx context.Context, id uuid.UUID, inactiveBy *uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&entity.DecisionRule{}).
		Where(`"ID" = ?`, id).
		Updates(map[string]any{
			`"STATUS"`:      enums.DecisionRuleStatusInactive,
			`"INACTIVE_BY"`: inactiveBy,
		}).Error
	if err != nil {
		return fmt.Errorf("deactivating decision rule: %w", err)
	}
	return nil
}

// DeleteDecisionRule soft-deletes a DecisionRule and all its child records in one transaction.
func (r *DecisionRuleWizardPostgresRepository) DeleteDecisionRule(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(`"DECISION_RULE_ID" = ?`, id).Delete(&entity.Schedule{}).Error; err != nil {
			return fmt.Errorf("deleting schedules: %w", err)
		}

		var ruleIDs []uuid.UUID
		if err := tx.Model(&entity.Rule{}).
			Where(`"DECISION_RULE_ID" = ?`, id).
			Pluck(`"ID"`, &ruleIDs).Error; err != nil {
			return fmt.Errorf("fetching rule ids: %w", err)
		}

		if len(ruleIDs) > 0 {
			if err := tx.Where(`"RULE_ID" IN ?`, ruleIDs).Delete(&entity.RuleAttribute{}).Error; err != nil {
				return fmt.Errorf("deleting rule attributes: %w", err)
			}
			if err := tx.Where(`"ID" IN ?`, ruleIDs).Delete(&entity.Rule{}).Error; err != nil {
				return fmt.Errorf("deleting rules: %w", err)
			}
		}

		if err := tx.Where(`"DECISION_RULE_ID" = ?`, id).Delete(&entity.RuleCondition{}).Error; err != nil {
			return fmt.Errorf("deleting rule conditions: %w", err)
		}

		if err := tx.Where(`"ID" = ?`, id).Delete(&entity.DecisionRule{}).Error; err != nil {
			return fmt.Errorf("deleting decision rule: %w", err)
		}
		return nil
	})
}

// FindConditionAttributesByDecisionRuleIDs returns leaf RuleConditions with Attribute
// preloaded for a batch of decision rule IDs. Leaf conditions are those whose
// ATTRIBUTE_ID is not the zero UUID (group nodes use uuid.Nil as a sentinel).
// A single LEFT JOIN replaces per-DR loops — no N+1.
func (r *DecisionRuleWizardPostgresRepository) FindConditionAttributesByDecisionRuleIDs(ctx context.Context, drIDs []uuid.UUID) ([]*entity.RuleCondition, error) {
	if len(drIDs) == 0 {
		return nil, nil
	}
	var conditions []*entity.RuleCondition
	err := r.db.WithContext(ctx).
		Joins("Attribute").
		Where(`rule_conditions."DECISION_RULE_ID" IN ?`, drIDs).
		Where(`rule_conditions."ATTRIBUTE_ID" != ?`, uuid.Nil).
		Find(&conditions).Error
	if err != nil {
		return nil, fmt.Errorf("finding condition attributes by decision rule ids: %w", err)
	}
	return conditions, nil
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
