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
	var results map[string]dto.ContentResult = make(map[string]dto.ContentResult)
	for _, sched := range schedules {
		if sched.DecisionRule == nil {
			continue
		}
		entries := BuildPlacementLogicEntries(*sched.DecisionRule, sched, placementName, nil)
		for _, entry := range entries {
			pass, err := EvaluateLogicConditions(entry.Conditions, userAttrs)
			if !pass || err != nil {
				continue
			}
			entry.LogicEval = pass
			// score is determined by the decision rule, so we can safely overwrite results for the same content path since they will have the same score and we want to avoid duplicate entries in the sorted results.
			if existing, exists := results[entry.ContentPath]; !exists || entry.Score > existing.Score {
				results[entry.ContentPath] = entry
			}
		}
	}
	sortedResults := make([]dto.ContentResult, 0, len(results))
	for _, entry := range results {
		sortedResults = append(sortedResults, entry)
	}
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].Score > sortedResults[j].Score
	})
	return sortedResults, nil
}
