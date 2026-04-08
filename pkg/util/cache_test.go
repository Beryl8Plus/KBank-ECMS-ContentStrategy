package util_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kbank-ecms/pkg/util"
)

// ---- minimal mock --------------------------------------------------------

type mockCache struct {
	// getSetFn overrides GetSet behaviour per test case.
	getSetFn func(ctx context.Context, key string, expiration time.Duration, loader func(context.Context) (string, error)) (string, error)
}

func (m *mockCache) Get(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockCache) Set(_ context.Context, _ string, _ string, _ time.Duration) error {
	return nil
}
func (m *mockCache) HGet(_ context.Context, _ string, _ string) (string, error) { return "", nil }
func (m *mockCache) HSet(_ context.Context, _ string, _ string, _ string) error { return nil }
func (m *mockCache) FlushDB(_ context.Context) error                            { return nil }
func (m *mockCache) GetSet(ctx context.Context, key string, expiration time.Duration, loader func(context.Context) (string, error)) (string, error) {
	return m.getSetFn(ctx, key, expiration, loader)
}

// ---- helpers -------------------------------------------------------------

type payload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

const testKey = "obj:1"
const testTTL = 5 * time.Minute

// ---- tests ---------------------------------------------------------------

// TestGetSet_CacheHit verifies that when GetSet returns a cached JSON value,
// the loader is NOT called and the value is correctly deserialised.
func TestGetSet_CacheHit(t *testing.T) {
	t.Parallel()

	loaderCalled := false
	cache := &mockCache{
		getSetFn: func(_ context.Context, _ string, _ time.Duration, _ func(context.Context) (string, error)) (string, error) {
			// Simulate a cache hit — return pre-serialised JSON directly.
			return `{"id":1,"name":"Alice"}`, nil
		},
	}

	got, err := util.GetSet(context.Background(), cache, testKey, testTTL, func(_ context.Context) (payload, error) {
		loaderCalled = true
		return payload{ID: 1, Name: "Alice"}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, payload{ID: 1, Name: "Alice"}, got)
	// The mock returned immediately; the loader should not have been invoked at
	// the util layer (the mock itself never calls it for a "hit").
	assert.False(t, loaderCalled)
}

// TestGetSet_CacheMiss_LoaderCalled verifies cache-miss path:
// the inner loader is called, the result serialised, and then returned.
func TestGetSet_CacheMiss_LoaderCalled(t *testing.T) {
	t.Parallel()

	loaderCalled := false
	cache := &mockCache{
		getSetFn: func(ctx context.Context, _ string, _ time.Duration, loader func(context.Context) (string, error)) (string, error) {
			// Simulate a cache miss by actually invoking the loader.
			return loader(ctx)
		},
	}

	want := payload{ID: 2, Name: "Bob"}
	got, err := util.GetSet(context.Background(), cache, testKey, testTTL, func(_ context.Context) (payload, error) {
		loaderCalled = true
		return want, nil
	})

	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.True(t, loaderCalled, "loader must be called on a cache miss")
}

// TestGetSet_LoaderError ensures that a loader failure is propagated without
// swallowing the error.
func TestGetSet_LoaderError(t *testing.T) {
	t.Parallel()

	loaderErr := errors.New("db unavailable")
	cache := &mockCache{
		getSetFn: func(ctx context.Context, _ string, _ time.Duration, loader func(context.Context) (string, error)) (string, error) {
			return loader(ctx) // call through to trigger loader error
		},
	}

	_, err := util.GetSet(context.Background(), cache, testKey, testTTL, func(_ context.Context) (payload, error) {
		return payload{}, loaderErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, loaderErr)
}

// TestGetSet_CacheGetError ensures that a hard cache error (not a miss) is
// returned directly to the caller.
func TestGetSet_CacheGetError(t *testing.T) {
	t.Parallel()

	cacheErr := errors.New("redis connection refused")
	cache := &mockCache{
		getSetFn: func(_ context.Context, _ string, _ time.Duration, _ func(context.Context) (string, error)) (string, error) {
			return "", cacheErr
		},
	}

	_, err := util.GetSet(context.Background(), cache, testKey, testTTL, func(_ context.Context) (payload, error) {
		return payload{}, nil
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, cacheErr)
}

// TestGetSet_MalformedJSON verifies that corrupted cached bytes cause an
// unmarshal error rather than a silent zero value.
func TestGetSet_MalformedJSON(t *testing.T) {
	t.Parallel()

	cache := &mockCache{
		getSetFn: func(_ context.Context, _ string, _ time.Duration, _ func(context.Context) (string, error)) (string, error) {
			return `{broken json`, nil // corrupted cache entry
		},
	}

	_, err := util.GetSet(context.Background(), cache, testKey, testTTL, func(_ context.Context) (payload, error) {
		return payload{}, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache unmarshal")
}
