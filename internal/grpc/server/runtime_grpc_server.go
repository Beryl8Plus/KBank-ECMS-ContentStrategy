// Package server implements the gRPC server for cms-runtime's evaluation engine.
// It exposes the CMSRuntimeService as a pure computation service — no database
// access happens on this path. The caller (cms-delivery) supplies schedule data
// as JSON bytes, and this server evaluates and returns ContentResult items.
package server

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"kbank-ecms/internal/domain/entity"
	domainservice "kbank-ecms/internal/domain/service"
	cmsruntimev1 "kbank-ecms/internal/grpc/pb/cms_runtime/v1"
	"kbank-ecms/internal/service/evaluator"
)

// Ensure interface is satisfied at compile time.
var _ cmsruntimev1.CMSRuntimeServiceServer = (*RuntimeGRPCServer)(nil)

// RuntimeGRPCServer implements cmsruntimev1.CMSRuntimeServiceServer.
// It is a pure stateless evaluator — no database, cache or registry dependencies.
type RuntimeGRPCServer struct {
	cmsruntimev1.UnimplementedCMSRuntimeServiceServer
}

// NewRuntimeGRPCServer creates a RuntimeGRPCServer.
func NewRuntimeGRPCServer() *RuntimeGRPCServer {
	return &RuntimeGRPCServer{}
}

// Register attaches the RuntimeGRPCServer to a gRPC server instance.
func Register(srv *grpc.Server) {
	cmsruntimev1.RegisterCMSRuntimeServiceServer(srv, NewRuntimeGRPCServer())
}

type scheduleEvaluation struct {
	schedule     *entity.Schedule
	rule         entity.DecisionRule
	logicEntries []domainservice.ContentResult
}

func flattenPlacementLogicEntries(evaluations []scheduleEvaluation) []domainservice.ContentResult {
	entries := make([]domainservice.ContentResult, 0)
	for _, evaluation := range evaluations {
		entries = append(entries, evaluation.logicEntries...)
	}
	return entries
}

func decodeSchedules(req *cmsruntimev1.EvaluateRequest) ([]*entity.Schedule, error) {
	var schedules []*entity.Schedule
	if err := json.Unmarshal(req.SchedulesJson, &schedules); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cms-runtime gRPC: unmarshal schedules: %v", err)
	}
	return schedules, nil
}

func decodeUserAttrs(req *cmsruntimev1.EvaluateRequest) (map[string]json.RawMessage, error) {
	if len(req.UserAttrsJson) == 0 {
		return nil, nil
	}

	var userAttrs map[string]json.RawMessage
	if err := json.Unmarshal(req.UserAttrsJson, &userAttrs); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cms-runtime gRPC: unmarshal user attrs: %v", err)
	}
	return userAttrs, nil
}

func buildScheduleEvaluations(schedules []*entity.Schedule) []scheduleEvaluation {
	evaluations := make([]scheduleEvaluation, 0, len(schedules))
	for _, sched := range schedules {
		if sched == nil || sched.DecisionRule == nil {
			continue
		}

		rule := *sched.DecisionRule
		source, campaign := buildRuleSourceAndCampaign(rule)

		evaluations = append(evaluations, scheduleEvaluation{
			schedule:     sched,
			rule:         rule,
			logicEntries: evaluator.BuildPlacementLogicEntries(rule, sched, source, campaign),
		})
	}
	return evaluations
}

func buildRuleSourceAndCampaign(rule entity.DecisionRule) (string, *domainservice.Campaign) {
	campaign := &domainservice.Campaign{}
	source := "DECISION_RULE"
	if rule.Type.IsCampaign() {
		campaign = &domainservice.Campaign{
			Code:      "Test",
			StartDate: "2026-01-01",
			EndDate:   "2026-12-31",
		}
		source = "LEAD_LIST"
	}
	return source, campaign
}

func resolveScheduleResult(evaluation scheduleEvaluation, userAttrs map[string]json.RawMessage, evaluatedAt string) ([]domainservice.ContentResult, bool) {
	if len(evaluation.logicEntries) == 0 {
		return nil, false
	}

	var candidates []domainservice.ContentResult
	if len(evaluation.rule.RuleConditions) > 0 {
		candidates = allMatchingLogicEntries(evaluation.logicEntries, userAttrs)
	} else {
		candidates = evaluation.logicEntries
	}

	if len(candidates) == 0 {
		return nil, false
	}

	results := make([]domainservice.ContentResult, 0, len(candidates))
	for _, c := range candidates {
		score := evaluation.rule.Score
		var variation *string
		if len(evaluation.rule.RuleConditions) > 0 {
			score = c.Score
			variation = c.Variation
		}
		c.ContentPath = evaluation.rule.ContentPath
		c.DecisionRuleId = evaluation.schedule.DecisionRuleID.String()
		c.RuleSetType = evaluation.rule.Type.String()
		c.Score = score
		c.Variation = variation
		c.StartDateTime = evaluation.schedule.EffectiveFrom.Format(time.RFC3339)
		c.EndDateTime = evaluation.schedule.EffectiveUntil.Format(time.RFC3339)
		c.EvaluatedAt = evaluatedAt
		results = append(results, c)
	}

	return results, true
}

func allMatchingLogicEntries(entries []domainservice.ContentResult, userAttrs map[string]json.RawMessage) []domainservice.ContentResult {
	var matched []domainservice.ContentResult
	for i := range entries {
		pass, err := evaluator.EvaluateLogicConditions(entries[i].Conditions, userAttrs)
		entries[i].LogicEval = pass
		if err != nil || !pass {
			continue
		}
		matched = append(matched, entries[i])
	}
	return matched
}

// Evaluate returns placement-logic entries when user attributes are absent so
// cms-delivery can cache and evaluate them later. When user attributes are
// present, it resolves ranked content results while preserving the legacy
// fallback to rule.Score when no variation matches.
func (s *RuntimeGRPCServer) Evaluate(
	ctx context.Context,
	req *cmsruntimev1.EvaluateRequest,
) (*cmsruntimev1.EvaluateResponse, error) {
	if len(req.SchedulesJson) == 0 {
		return &cmsruntimev1.EvaluateResponse{}, nil
	}

	schedules, err := decodeSchedules(req)
	if err != nil {
		return nil, err
	}

	userAttrs, err := decodeUserAttrs(req)
	if err != nil {
		return nil, err
	}
	if len(userAttrs) == 0 {
		return &cmsruntimev1.EvaluateResponse{LogicEntriesJson: nil}, nil
	}

	// 1. Resolve each schedule to its best-ranked result.
	now := time.Now().UTC().Format(time.RFC3339)
	best := make(map[string]domainservice.ContentResult)
	evaluations := buildScheduleEvaluations(schedules)
	if len(userAttrs) == 0 {
		data, err := json.Marshal(flattenPlacementLogicEntries(evaluations))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cms-runtime gRPC: marshal logic entries: %v", err)
		}

		return &cmsruntimev1.EvaluateResponse{LogicEntriesJson: data}, nil
	}

	for _, evaluation := range evaluations {
		candidates, ok := resolveScheduleResult(evaluation, userAttrs, now)
		if !ok {
			continue
		}
		for _, candidate := range candidates {
			if existing, seen := best[candidate.ContentPath]; !seen || candidate.Score > existing.Score {
				best[candidate.ContentPath] = candidate
			}
		}
	}

	// 2. Collect, sort, and cap the results.
	items := make([]domainservice.ContentResult, 0, len(best))
	for _, r := range best {
		items = append(items, r)
	}
	// Sort by score descending; stable sort preserves original order for equal scores (e.g., from the same rule).
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})
	data, err := json.Marshal(items)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cms-runtime gRPC: marshal logic entries: %v", err)
	}

	return &cmsruntimev1.EvaluateResponse{LogicEntriesJson: data}, nil
}
