// Package evaluator provides pure-function rule evaluation utilities:
// EvaluateRuleScore, EvaluateLogicConditions, GenerateConditionHash,
// and BuildPlacementLogicEntry.
package evaluator

import (
	"context"
	"encoding/json"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// RuleEvaluator computes a score for a single DecisionRule.
// Each evaluator handles exactly one rule type.
//
// userAttrs carries live user attribute values (attributeID → compact JSON value).
// When non-nil, leaf conditions compare against these values.
// When nil, conditions with user-dependent attributes are treated as non-match
// (the caller is expected to defer real evaluation to delivery time).
type RuleEvaluator interface {
	// Evaluate returns the computed score for the given rule.
	Evaluate(ctx context.Context, rule entity.DecisionRule, userAttrs map[string]json.RawMessage) (*string, float64, error)

	// RuleType returns the decision rule type string this evaluator handles.
	RuleType() enums.EvaluateType
}

// Registry maps rule type strings to their corresponding RuleEvaluator.
type Registry struct {
	evaluators map[enums.EvaluateType]RuleEvaluator
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		evaluators: make(map[enums.EvaluateType]RuleEvaluator),
	}
}

// Register adds the given evaluator to the registry keyed by its RuleType().
// If an evaluator for the same type already exists it is overwritten.
func (r *Registry) Register(e RuleEvaluator) {
	r.evaluators[e.RuleType()] = e
}

// Get returns the RuleEvaluator registered for ruleType and whether it was found.
func (r *Registry) Get(ruleType enums.EvaluateType) (RuleEvaluator, bool) {
	e, ok := r.evaluators[ruleType]
	return e, ok
}
