package util

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainrepo "kbank-ecms/internal/domain/repository"
)

// GetSet is a generic, type-safe cache-aside helper.
//
// It tries to fetch a JSON-encoded value for key from cache. On a miss it calls
// loader, JSON-encodes the result, stores it with the given expiration, then
// returns the decoded value.
//
// Usage:
//
//	user, err := util.GetSet(ctx, cache, "user:42", 5*time.Minute, func(ctx context.Context) (*User, error) {
//	    return db.FindUser(ctx, 42)
//	})
func GetSet[T any](
	ctx context.Context,
	cache domainrepo.CacheRepository,
	key string,
	expiration time.Duration,
	loader func(ctx context.Context) (T, error),
) (T, error) {
	var zero T

	// Wrap loader so it returns a JSON string — matches the CacheRepository contract.
	stringLoader := func(ctx context.Context) (string, error) {
		val, err := loader(ctx)
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("cache marshal for key %q: %w", key, err)
		}
		return string(b), nil
	}

	raw, err := cache.GetSet(ctx, key, expiration, stringLoader)
	if err != nil {
		return zero, err
	}

	var result T
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return zero, fmt.Errorf("cache unmarshal for key %q: %w", key, err)
	}
	return result, nil
}
