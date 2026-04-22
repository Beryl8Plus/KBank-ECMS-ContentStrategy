// Package client provides a gRPC client wrapper for the cms-runtime evaluation service.
// It implements domain/service.RuntimeEvaluator so that CMSDeliveryService is decoupled
// from the concrete transport.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	domainservice "kbank-ecms/internal/domain/service"
	cmsruntimev1 "kbank-ecms/internal/grpc/pb/cms_runtime/v1"
	"kbank-ecms/pkg/ctxconsts"
)

// defaultTimeout is the maximum time allowed for a single cms-runtime RPC.
const defaultTimeout = 3 * time.Second

// compile-time interface guard.
var _ domainservice.RuntimeEvaluator = (*RuntimeGRPCClient)(nil)

// RuntimeGRPCClient implements domainservice.RuntimeEvaluator over gRPC.
// It is safe for concurrent use.
type RuntimeGRPCClient struct {
	conn    *grpc.ClientConn
	client  cmsruntimev1.CMSRuntimeServiceClient
	timeout time.Duration
}

func newRuntimeGRPCClient(addr string, opts ...grpc.DialOption) (*RuntimeGRPCClient, error) {
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("cms-delivery gRPC dial %s: %w", addr, err)
	}
	return &RuntimeGRPCClient{
		conn:    conn,
		client:  cmsruntimev1.NewCMSRuntimeServiceClient(conn),
		timeout: defaultTimeout,
	}, nil
}

// NewRuntimeGRPCClient dials addr (e.g. "cms-runtime:50051") and returns a
// ready-to-use client. The caller must call Close() when done.
func NewRuntimeGRPCClient(addr string) (*RuntimeGRPCClient, error) {
	return newRuntimeGRPCClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(correlationIDUnaryInterceptor),
	)
}

// correlationIDUnaryInterceptor reads the correlation ID from the context
// (stored by the HTTP logger middleware) and forwards it to the server as
// the "requestID" gRPC metadata header.
func correlationIDUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if id := ctxconsts.GetCorrelationID(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "requestID", id)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

// buildEvaluateRequest converts domain types to the native proto request.
// Schedules are still JSON-serialised (complex entity graph) but user attrs
// use the native map<string,bytes> field to avoid JSON marshalling.
func (c *RuntimeGRPCClient) buildEvaluateRequest(
	placementName string,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) (*cmsruntimev1.EvaluateRequest, error) {
	schedulesJSON, err := json.Marshal(schedules)
	if err != nil {
		return nil, fmt.Errorf("cms-delivery: marshal schedules for gRPC: %w", err)
	}

	// Convert map[string]json.RawMessage → map[string][]byte (zero-copy).
	protoAttrs := make(map[string][]byte, len(userAttrs))
	for k, v := range userAttrs {
		protoAttrs[k] = []byte(v)
	}

	return &cmsruntimev1.EvaluateRequest{
		PlacementName: placementName,
		SchedulesJson: schedulesJSON,
		UserAttrs:     protoAttrs,
	}, nil
}

func (c *RuntimeGRPCClient) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.timeout)
}

// Close releases the underlying gRPC connection.
func (c *RuntimeGRPCClient) Close() error {
	return c.conn.Close()
}

// Evaluate serialises schedules to JSON, sends them to cms-runtime via the
// Evaluate RPC, and returns []ContentResult for per-user evaluation.
// Returns nil, nil on an empty response (no rules matched).
func (c *RuntimeGRPCClient) Evaluate(
	ctx context.Context,
	placementName string,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) ([]dto.ContentResult, error) {
	req, err := c.buildEvaluateRequest(placementName, schedules, userAttrs)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	resp, err := c.client.Evaluate(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("cms-delivery gRPC Evaluate(%s): %w", placementName, err)
	}

	// Prefer native proto results; fall back to legacy JSON bytes.
	if len(resp.Results) > 0 {
		return protoResultsToDomain(resp.Results), nil
	}

	// Legacy fallback: read JSON bytes.
	if len(resp.LogicEntriesJson) == 0 {
		return nil, nil
	}
	var entries []dto.ContentResult
	if err := json.Unmarshal(resp.LogicEntriesJson, &entries); err != nil {
		return nil, fmt.Errorf("cms-delivery gRPC: unmarshal logic entries: %w", err)
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Proto ↔ Domain converters
// ---------------------------------------------------------------------------

// protoResultsToDomain converts repeated proto ContentResult to domain DTOs.
func protoResultsToDomain(pbResults []*cmsruntimev1.ContentResult) []dto.ContentResult {
	results := make([]dto.ContentResult, 0, len(pbResults))
	for _, pb := range pbResults {
		if pb == nil {
			continue
		}
		r := dto.ContentResult{
			DecisionRuleId: pb.DecisionRuleId,
			ContentPath:    pb.ContentPath,
			RuleSetType:    pb.RuleSetType,
			Source:         pb.Source,
			Score:          pb.Score,
			StartDateTime:  pb.StartDateTime,
			EndDateTime:    pb.EndDateTime,
			LogicHash:      pb.LogicHash,
			LogicExpr:      pb.LogicExpr,
			LogicEval:      pb.LogicEval,
		}
		if pb.Variation != nil {
			v := *pb.Variation
			r.Variation = &v
		}
		if pb.Campaign != nil {
			r.Campaign = &dto.Campaign{
				Code:      pb.Campaign.Code,
				StartDate: pb.Campaign.StartDate,
				EndDate:   pb.Campaign.EndDate,
			}
		}
		for _, lc := range pb.Conditions {
			r.Conditions = append(r.Conditions, dto.LogicCondition{
				ConditionID:       lc.ConditionId,
				ParentConditionID: lc.ParentConditionId,
				AttributeID:       lc.AttributeId,
				DataType:          lc.DataType,
				LogicalOperator:   lc.LogicalOperator,
				ConnectorOperator: lc.ConnectorOperator,
				Sequence:          int(lc.Sequence),
				ExpectedValue:     json.RawMessage(lc.ExpectedValue),
			})
		}
		results = append(results, r)
	}
	return results
}
