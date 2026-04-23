package evaluator

import (
	"context"
	"encoding/json"
	"sort"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
)

// LocalEvaluator implements service.RuntimeEvaluator using the internal
// condition evaluator — no network hop required.
type LocalEvaluator struct{}

// NewLocalEvaluator returns a new LocalEvaluator.
func NewLocalEvaluator() *LocalEvaluator {
	return &LocalEvaluator{}
}

// Evaluate builds ContentResult entries for each schedule's decision rule,
// evaluates conditions against userAttrs, and returns entries sorted by score
// descending. LogicEval is set true when conditions pass for a given entry.
func (e *LocalEvaluator) Evaluate(
	_ context.Context,
	placementName string,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) ([]dto.ContentResult, error) {
	var results []dto.ContentResult
	for _, sched := range schedules {
		if sched.DecisionRule == nil {
			continue
		}
		entries := BuildPlacementLogicEntries(*sched.DecisionRule, sched, placementName, nil)
		for _, entry := range entries {
			pass, err := EvaluateLogicConditions(entry.Conditions, userAttrs)
			if err != nil {
				continue
			}
			entry.LogicEval = pass
			results = append(results, entry)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results, nil
}
