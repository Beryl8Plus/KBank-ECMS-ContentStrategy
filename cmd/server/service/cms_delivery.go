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
	// customerInfo contains the Customer Information System channel and user identifiers; it is used to
	// scope the personalized placement cache key.
	// userAttrs maps attribute UUID strings to compact JSON values (the live
	// attribute values for this user, as received in the request payload).
	// Results are cached at cms:placement:{cisID}:{name} for resultTTL.
	GetPersonalizedContent(
		ctx context.Context,
		customerInfo *dto.CustomerRequest,
		placementNames []string,
	) ([]dto.ContentResult, error)

	// FlushCache removes the cached results for the given placement names.
	// If placementNames is non-empty, only those placements are evicted.
	// If placementNames is nil or empty, ALL placement caches are flushed.
	FlushCache(ctx context.Context, placementNames []string, isEvaluate bool) error

	// GetCacheKeys returns the list of cache keys for the given placement names.
	// This is used for monitoring and debugging purposes to see which cache keys are currently stored in Memory.
	GetCacheKeys(ctx context.Context) ([]string, error)

	// GetCacheValue returns the cached value for the given key. This is used for monitoring and debugging purposes to inspect the contents of specific cache entries.
	GetCacheValue(ctx context.Context, key string) (json.RawMessage, error)

	// GetCacheStatus returns whether the cache is under memory pressure and the current memory usage percentage.
	GetCacheStatus(ctx context.Context) (isMemPressure bool, memoryUsagePct float64, err error)
}
