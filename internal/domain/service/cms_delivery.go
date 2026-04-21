package service

import (
	"context"
	"encoding/json"
	"kbank-ecms/internal/delivery/http/dto"
)

// DeliveryService defines the cms-delivery contract.
// Responsible for serving pre-computed content results from Redis.
// This service has no PostgreSQL dependency; all reads come from the Redis cache.
type DeliveryService interface {
	// GetPersonalizedContent evaluates placement logic entries against the
	// caller's user attributes and returns ranked ContentResult items.
	// cisID identifies the Customer Information System channel; it is used to
	// scope the personalized placement cache key.
	// userAttrs maps attribute UUID strings to compact JSON values (the live
	// attribute values for this user, as received in the request payload).
	// Results are cached at cms:placement:{cisID}:{name} for resultTTL.
	GetPersonalizedContent(
		ctx context.Context,
		cisID, userID string,
		placementNames []string,
		userAttrs map[string]json.RawMessage,
	) ([]dto.ContentResult, error)

	// FlushCache removes the cached results for the given placement names.
	// If placementNames is non-empty, only those placements are evicted.
	// If placementNames is nil or empty, ALL placement caches are flushed.
	FlushCache(ctx context.Context, placementNames []string, isEvaluate bool) error
}
