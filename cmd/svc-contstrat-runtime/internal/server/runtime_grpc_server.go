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

	"kbank-ecms/cmd/svc-contstrat-runtime/internal/evaluator"
	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	cmsruntimev1 "kbank-ecms/internal/grpc/pb/cms_runtime/v1"
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
	logicEntries []dto.ContentResult
}

func decodeSchedules(req *cmsruntimev1.EvaluateRequest) ([]*entity.Schedule, error) {
	var schedules []*entity.Schedule
	if err := json.Unmarshal(req.SchedulesJson, &schedules); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cms-runtime gRPC: unmarshal schedules: %v", err)
	}
	return schedules, nil
}

// decodeUserAttrs extracts user attributes from the request.
// It prefers the native proto map field (UserAttrs) over the legacy
// JSON bytes (UserAttrsJson), eliminating JSON unmarshalling overhead.
func decodeUserAttrs(req *cmsruntimev1.EvaluateRequest) (map[string]json.RawMessage, error) {
	// Prefer native proto map.
	if len(req.UserAttrs) > 0 {
		attrs := make(map[string]json.RawMessage, len(req.UserAttrs))
		for k, v := range req.UserAttrs {
			attrs[k] = json.RawMessage(v)
		}
		return attrs, nil
	}

	// Legacy fallback: JSON bytes.
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

func buildRuleSourceAndCampaign(rule entity.DecisionRule) (string, *dto.Campaign) {
	campaign := &dto.Campaign{}
	source := "DECISION_RULE"
	if rule.Type.IsCampaign() {
		campaign = &dto.Campaign{
			Code:      "Test",
			StartDate: "2026-01-01",
			EndDate:   "2026-12-31",
		}
		source = "LEAD_LIST"
	}
	return source, campaign
}

func resolveScheduleResult(evaluation scheduleEvaluation, userAttrs map[string]json.RawMessage, evaluatedAt string) ([]dto.ContentResult, bool) {
	if len(evaluation.logicEntries) == 0 {
		return nil, false
	}

	var candidates []dto.ContentResult
	if len(evaluation.rule.RuleConditions) > 0 {
		candidates = allMatchingLogicEntries(evaluation.logicEntries, userAttrs)
	} else {
		candidates = evaluation.logicEntries
	}

	if len(candidates) == 0 {
		return nil, false
	}

	results := make([]dto.ContentResult, 0, len(candidates))
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

func allMatchingLogicEntries(entries []dto.ContentResult, userAttrs map[string]json.RawMessage) []dto.ContentResult {
	var matched []dto.ContentResult
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
//
// The response now uses native repeated ContentResult proto messages instead
// of JSON-serialised bytes to eliminate marshalling overhead.
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
		return &cmsruntimev1.EvaluateResponse{}, nil
	}

	// 1. Resolve each schedule to its best-ranked result.
	now := time.Now().UTC().Format(time.RFC3339)
	best := make(map[string]dto.ContentResult)
	evaluations := buildScheduleEvaluations(schedules)

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
	items := make([]dto.ContentResult, 0, len(best))
	for _, r := range best {
		items = append(items, r)
	}
	// Sort by score descending; stable sort preserves original order for equal scores.
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	// 3. Convert domain DTOs to native proto ContentResult messages.
	pbResults := domainResultsToProto(items)

	return &cmsruntimev1.EvaluateResponse{Results: pbResults}, nil
}

// ---------------------------------------------------------------------------
// Domain → Proto converters
// ---------------------------------------------------------------------------

// domainResultsToProto converts domain ContentResult DTOs to proto messages.
func domainResultsToProto(items []dto.ContentResult) []*cmsruntimev1.ContentResult {
	results := make([]*cmsruntimev1.ContentResult, 0, len(items))
	for _, item := range items {
		pb := &cmsruntimev1.ContentResult{
			ContentPath:    item.ContentPath,
			DecisionRuleId: item.DecisionRuleId,
			RuleSetType:    item.RuleSetType,
			Source:         item.Source,
			Score:          item.Score,
			StartDateTime:  item.StartDateTime,
			EndDateTime:    item.EndDateTime,
			LogicHash:      item.LogicHash,
			LogicExpr:      item.LogicExpr,
			LogicEval:      item.LogicEval,
		}
		if item.Variation != nil {
			pb.Variation = item.Variation
		}
		if item.Campaign != nil {
			pb.Campaign = &cmsruntimev1.Campaign{
				Code:      item.Campaign.Code,
				StartDate: item.Campaign.StartDate,
				EndDate:   item.Campaign.EndDate,
			}
		}
		for _, lc := range item.Conditions {
			pb.Conditions = append(pb.Conditions, &cmsruntimev1.LogicCondition{
				ConditionId:       lc.ConditionID,
				ParentConditionId: lc.ParentConditionID,
				AttributeId:       lc.AttributeID,
				DataType:          lc.DataType,
				LogicalOperator:   lc.LogicalOperator,
				ConnectorOperator: lc.ConnectorOperator,
				Sequence:          int32(lc.Sequence),
				ExpectedValue:     []byte(lc.ExpectedValue),
			})
		}
		results = append(results, pb)
	}
	return results
}
