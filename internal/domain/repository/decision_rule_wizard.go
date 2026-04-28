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

	// UpdateStep1 atomically updates a DecisionRule header, upserts conditions, and cascades
	// deletes into Step 2 (Rules + RuleAttributes) for removed conditions.
	// Returns the IDs of Rules that were deleted as a cascade side-effect.
	UpdateStep1(ctx context.Context, id uuid.UUID, dr *entity.DecisionRule, toUpsert []*entity.RuleCondition, toDeleteConditionIDs []uuid.UUID) ([]uuid.UUID, error)

	// FindDecisionRuleByID returns a DecisionRule by primary key, or (nil, nil) when not found.
	FindDecisionRuleByID(ctx context.Context, id uuid.UUID) (*entity.DecisionRule, error)

	// FindTemplateConditions returns all RuleConditions for a DecisionRule with Attribute preloaded.
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

	// SaveStep2 atomically deletes removed rules, upserts remaining rules, and replaces their RuleAttributes.
	// toDeleteRuleIDs: rules that exist in DB but are absent from the current request.
	SaveStep2(ctx context.Context, rules []*entity.Rule, attrs []*entity.RuleAttribute, toDeleteRuleIDs []uuid.UUID) error

	// --- Step 3: Schedules ---

	// CountSchedulesByPlacementExcludingDR counts active schedules for a placement
	// that belong to OTHER decision rules (excludes the given drID).
	// Used to enforce the per-placement schedule cap before a full replacement.
	CountSchedulesByPlacementExcludingDR(ctx context.Context, placementID, excludeDRID uuid.UUID) (int64, error)

	// SaveStep3 atomically replaces all schedules for a DecisionRule:
	// deletes every existing schedule for the DR then inserts the new set.
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

	// FindConditionAttributesByDecisionRuleIDs returns leaf RuleConditions with Attribute
	// preloaded for the given DR IDs. Leaf = conditions with a real AttributeID (not the
	// zero UUID used for group nodes). One query for all DRs — prevents N+1 on the list endpoint.
	FindConditionAttributesByDecisionRuleIDs(ctx context.Context, drIDs []uuid.UUID) ([]*entity.RuleCondition, error)

	// FindSchedulesByDecisionRuleID returns Schedules (with Placement and Channel preloaded)
	// for a single decision rule, or nil when the rule has no schedules.
	FindSchedulesByDecisionRuleID(ctx context.Context, decisionRuleID uuid.UUID) ([]*entity.Schedule, error)

	// --- Lifecycle ---

	// CloneDecisionRule atomically inserts a deep-copied DecisionRule with its
	// conditions, rules, rule attributes, and placeholder schedules (placement-only,
	// time fields zeroed so the user must re-select dates in Step 3).
	CloneDecisionRule(ctx context.Context, dr *entity.DecisionRule, conditions []*entity.RuleCondition, rules []*entity.Rule, attrs []*entity.RuleAttribute, schedules []*entity.Schedule) error

	// DeactivateDecisionRule sets a DecisionRule status to INACTIVE and records who did it.
	DeactivateDecisionRule(ctx context.Context, id uuid.UUID, inactiveBy *uuid.UUID) error

	// DeleteDecisionRule soft-deletes a DecisionRule and all its child records
	// (schedules, rule attributes, rules, rule conditions) within a single transaction.
	DeleteDecisionRule(ctx context.Context, id uuid.UUID) error
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
