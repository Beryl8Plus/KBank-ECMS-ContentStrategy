package repository

import (
	"context"
	"time"
)

// RedisCacheRepository defines the contract for cache operations.
// Redis functions are used as a reference for method signatures and expected behaviors.
type RedisCacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	HGet(ctx context.Context, key string, field string) (string, error)
	HSet(ctx context.Context, key string, field string, value string) error
	FlushDB(ctx context.Context) error

	// GetSet retrieves the cached string for key. On a cache miss it calls
	// loader to produce the value, stores it with the given expiration, then
	// returns it. A cache-miss error (e.g. redis.Nil) is NOT propagated.
	GetSet(ctx context.Context, key string, expiration time.Duration, loader func(ctx context.Context) (string, error)) (string, error)

	// Delete removes the value stored at key.
	// Returns nil if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Subscribe listens to a Redis channel and returns a channel for messages.
	Subscribe(ctx context.Context, channel string) (<-chan string, error)
}
