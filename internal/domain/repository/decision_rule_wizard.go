package repository

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// DecisionRuleWizardRepository defines the data-access contract for the 4-step creation wizard.
type DecisionRuleWizardRepository interface {
	// --- Step 1: Create draft ---

	// SaveStep1 atomically creates a DecisionRule and all its template RuleConditions.
	SaveStep1(ctx context.Context, dr *entity.DecisionRule, conditions []*entity.RuleCondition) error

	// FindDecisionRuleByID returns a DecisionRule by primary key, or (nil, nil) when not found.
	FindDecisionRuleByID(ctx context.Context, id uuid.UUID) (*entity.DecisionRule, error)

	// FindTemplateConditions returns all RuleConditions for a DecisionRule that are
	// template rows (no Rule association), with Attribute preloaded.
	FindTemplateConditions(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.RuleCondition, error)

	// --- Step 2: Rule sets ---

	// FindConditionsByIDs returns RuleConditions matching the given IDs.
	FindConditionsByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.RuleCondition, error)

	// FindRulesByDecisionRuleID returns all Rules for a DecisionRule ordered by order_no.
	FindRulesByDecisionRuleID(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.Rule, error)

	// FindRuleAttributesByRuleIDs returns all RuleAttributes for the given rule IDs,
	// with Attribute preloaded.
	FindRuleAttributesByRuleIDs(ctx context.Context, ruleIDs []uuid.UUID) ([]*entity.RuleAttribute, error)

	// FindRuleByID returns a Rule by primary key, or (nil, nil) when not found.
	FindRuleByID(ctx context.Context, id uuid.UUID) (*entity.Rule, error)

	// SaveStep2 atomically upserts Rules and their RuleAttributes.
	// For each entry: if Rule.ID is zero, it is inserted; otherwise it is updated.
	// Existing RuleAttributes for each rule_id are replaced.
	SaveStep2(ctx context.Context, rules []*entity.Rule, attrs []*entity.RuleAttribute) error

	// --- Step 3: Schedules ---

	// CountSchedulesByPlacement counts active (non-deleted) schedules for a placement.
	CountSchedulesByPlacement(ctx context.Context, placementID uuid.UUID) (int64, error)

	// ExistsSchedule reports whether an active schedule already exists for the
	// (decisionRuleID, placementID) pair.
	ExistsSchedule(ctx context.Context, decisionRuleID, placementID uuid.UUID) (bool, error)

	// SaveStep3 atomically inserts schedules for a DecisionRule.
	SaveStep3(ctx context.Context, decisionRuleID uuid.UUID, schedules []*entity.Schedule) error

	// --- Step 4: Activate ---

	// ActivateDecisionRule sets the DecisionRule status to ACTIVE.
	ActivateDecisionRule(ctx context.Context, decisionRuleID uuid.UUID) error

	// --- List ---

	// ListDecisionRules returns a paginated list of DecisionRules matching the filter.
	ListDecisionRules(ctx context.Context, f DecisionRuleListFilter) ([]*entity.DecisionRule, int64, error)

	// FindSchedulesWithPlacementsByDecisionRuleIDs returns Schedules (with Placement and Channel
	// preloaded) for the given decision rule IDs.
	FindSchedulesWithPlacementsByDecisionRuleIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Schedule, error)

	// FindSchedulesByDecisionRuleID returns Schedules (with Placement and Channel preloaded)
	// for a single decision rule, or nil when the rule has no schedules.
	FindSchedulesByDecisionRuleID(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.Schedule, error)
}

// DecisionRuleListFilter holds the query parameters for the list endpoint.
type DecisionRuleListFilter struct {
	Type         string
	EvaluateType string
	Status       string
	Keyword      string
	Page         int
	Limit        int
}
