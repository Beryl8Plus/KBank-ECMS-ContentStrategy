package evaluator

import (
	"context"
	"encoding/json"
	"sort"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
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
	leads []entity.Lead,
) ([]dto.ContentResult, error) {
	// var results = make(map[string]dto.ContentResult)
	var results = make([]dto.ContentResult, 0)
	for _, sched := range schedules {
		if sched.DecisionRule == nil {
			continue
		}
		entries := BuildPlacementLogicEntries(*sched.DecisionRule, sched, placementName, nil)
		isSalesTarget := sched.DecisionRule.Type == enums.DecisionTypeSalesTarget
		for _, entry := range entries {
			pass, err := EvaluateLogicConditions(entry.Conditions, userAttrs)
			if !pass || err != nil {
				continue
			}
			entry.LogicEval = pass

			// SALES_TARGET rules expand into one entry per lead that targets
			// this placement. The rule's own entry is dropped when no lead
			// matches — a sales-targeted rule without a lead means no offer.
			if isSalesTarget {
				results = append(results, expandWithLeads(entry, leads, placementName, entry.Score)...)
				continue
			}

			// score is determined by the decision rule, so we can safely overwrite results for the same content path since they will have the same score and we want to avoid duplicate entries in the sorted results.
			// if existing, exists := results[entry.ContentPath]; !exists || entry.Score > existing.Score {
			// 	results[entry.ContentPath] = entry
			// }
			results = append(results, entry)
		}
	}

	sortedResults := make([]dto.ContentResult, 0, len(results))
	sortedResults = append(sortedResults, results...)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].Score > sortedResults[j].Score
	})
	return sortedResults, nil
}
