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

func (c *RuntimeGRPCClient) buildEvaluateRequest(
	placementName string,
	schedules []*entity.Schedule,
	userAttrs map[string]json.RawMessage,
) (*cmsruntimev1.EvaluateRequest, error) {
	schedulesJSON, err := json.Marshal(schedules)
	if err != nil {
		return nil, fmt.Errorf("cms-delivery: marshal schedules for gRPC: %w", err)
	}

	var userAttrsJSON []byte
	if len(userAttrs) > 0 {
		userAttrsJSON, err = json.Marshal(userAttrs)
		if err != nil {
			return nil, fmt.Errorf("cms-delivery: marshal user attrs for gRPC: %w", err)
		}
	}

	return &cmsruntimev1.EvaluateRequest{
		PlacementName: placementName,
		SchedulesJson: schedulesJSON,
		UserAttrsJson: userAttrsJSON,
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

	if len(resp.LogicEntriesJson) == 0 {
		return nil, nil
	}

	var entries []dto.ContentResult
	if err := json.Unmarshal(resp.LogicEntriesJson, &entries); err != nil {
		return nil, fmt.Errorf("cms-delivery gRPC: unmarshal logic entries: %w", err)
	}
	return entries, nil
}
