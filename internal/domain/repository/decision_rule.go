package repository

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// DecisionRuleRepository defines the contract for decision-rule database operations.
type DecisionRuleRepository interface {
	// GetDecisionRuleByScheduleID retrieves the DecisionRule associated with the given
	// schedule ID, preloaded with RuleConditions, Attributes, Rules, and RuleAttributes.
	// Returns (nil, nil) when no matching schedule or decision rule is found.
	GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error)

	// GetDecisionRuleByScheduleIDs retrieves the DecisionRules associated with the given
	// schedule IDs, preloaded with RuleConditions, Attributes, Rules, and RuleAttributes.
	// Returns a map of scheduleID to DecisionRule. ScheduleIDs with no matching decision rule are omitted.
	GetDecisionRuleByScheduleIDs(ctx context.Context, scheduleIDs []uuid.UUID) (map[uuid.UUID]*entity.DecisionRule, error)
}
