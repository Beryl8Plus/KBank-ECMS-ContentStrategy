package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCache creates a CacheMemory with a unique namespace to avoid
// Prometheus metric registration conflicts between parallel tests.
func newTestCache(t *testing.T) *CacheMemory[string] {
	t.Helper()
	c := NewCacheMemory[string](t.Name(), 0.99, time.Hour)
	t.Cleanup(func() { c.Stop() })
	return c
}

// TestGet_ReturnsHitBeforeExpiry verifies that a valid (non-expired) entry is returned.
func TestGet_ReturnsHitBeforeExpiry(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "value", 10*time.Second)

	got, ok := c.Get("key")
	require.True(t, ok, "expected cache hit")
	assert.Equal(t, "value", got)
}

// TestGet_ReturnsMissAfterExpiry verifies that an expired entry is not returned.
func TestGet_ReturnsMissAfterExpiry(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "value", 1*time.Millisecond)

	time.Sleep(5 * time.Millisecond) // let TTL elapse

	got, ok := c.Get("key")
	assert.False(t, ok, "expected cache miss after TTL expiry")
	assert.Empty(t, got)
}

// TestGet_MissOnNonExistentKey verifies a miss for a key that was never stored.
func TestGet_MissOnNonExistentKey(t *testing.T) {
	c := newTestCache(t)
	_, ok := c.Get("ghost")
	assert.False(t, ok)
}

// TestSet_RejectsUnderMemPressure verifies that entries are not stored when
// memory pressure is active.
func TestSet_RejectsUnderMemPressure(t *testing.T) {
	c := newTestCache(t)

	c.mu.Lock()
	c.isMemPressure = true
	c.mu.Unlock()

	c.Set("key", "value", 10*time.Second)

	_, ok := c.Get("key")
	assert.False(t, ok, "entry should not be stored under memory pressure")
}

// TestPurgeExpired_RemovesOnlyExpiredKeys verifies that purgeExpired removes
// expired entries while leaving valid entries intact.
func TestPurgeExpired_RemovesOnlyExpiredKeys(t *testing.T) {
	c := newTestCache(t)
	c.Set("live", "live-val", 10*time.Second)
	c.Set("dead", "dead-val", 1*time.Millisecond)

	time.Sleep(5 * time.Millisecond) // let "dead" TTL elapse

	c.purgeExpired()

	_, deadOk := c.cache.Get("dead")
	_, liveOk := c.cache.Get("live")

	assert.False(t, deadOk, "expired entry 'dead' should have been purged")
	assert.True(t, liveOk, "valid entry 'live' should still be present")
}

// TestDelete_RemovesKey verifies that Delete removes the specified key.
func TestDelete_RemovesKey(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "value", 10*time.Second)
	c.Delete("key")

	_, ok := c.Get("key")
	assert.False(t, ok)
}

// TestClear_RemovesAllKeys verifies that Clear empties the entire cache.
func TestClear_RemovesAllKeys(t *testing.T) {
	c := newTestCache(t)
	c.Set("a", "1", 10*time.Second)
	c.Set("b", "2", 10*time.Second)
	c.Clear()

	assert.Equal(t, 0, c.cache.ItemCount(), "cache should be empty after Clear")
}
