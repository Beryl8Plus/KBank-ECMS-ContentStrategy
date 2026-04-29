package service

import (
	"context"
	"encoding/json"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
)

// RuntimeEvaluator defines the contract for on-demand rule evaluation.
// Used by cms-delivery to call cms-runtime via gRPC on cache miss.
// The caller (cms-delivery) is responsible for querying active schedules
// from the database before calling this interface.
type RuntimeEvaluator interface {
	// Evaluate sends the provided schedules to the cms-runtime evaluation
	// engine and returns ranked ContentResult entries for the specified
	// placement. If maxResults ≤ 0, a default of 10 is used. If the remote
	// call succeeds but produces no entries, the method returns (nil, nil).
	// Returned ContentResult items include rule conditions and expected
	// per-user attribute values used at delivery time. leads carries
	// CLEN-fetched lead offers (already filtered to the customer); rules
	// of type SALES_TARGET expand into one entry per matching lead.
	Evaluate(
		ctx context.Context,
		placementName string,
		schedules []*entity.Schedule,
		userAttrs map[string]json.RawMessage,
		leads []entity.Lead,
	) ([]dto.ContentResult, error)
}
