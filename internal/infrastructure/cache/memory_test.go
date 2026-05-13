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

// TestSet_RejectsNewEntryUnderMemPressure verifies that a brand-new key is not
// stored when memory pressure is active (avoids growing memory footprint).
func TestSet_RejectsNewEntryUnderMemPressure(t *testing.T) {
	c := newTestCache(t)

	c.mu.Lock()
	c.isMemPressure = true
	c.mu.Unlock()

	c.Set("key", "value", 10*time.Second)

	_, ok := c.Get("key")
	assert.False(t, ok, "new entry should not be stored under memory pressure")
}

// TestSet_UpdatesExistingEntryUnderMemPressure verifies that an already-cached
// key is refreshed (net-zero memory change) even when memory pressure is active.
// This allows the evaluate background tick to keep L1 data fresh under pressure
// without growing the total cache footprint.
func TestSet_UpdatesExistingEntryUnderMemPressure(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "original", 10*time.Second)

	c.mu.Lock()
	c.isMemPressure = true
	c.mu.Unlock()

	c.Set("key", "refreshed", 10*time.Second)

	got, ok := c.Get("key")
	require.True(t, ok, "existing entry must be refreshable under memory pressure")
	assert.Equal(t, "refreshed", got, "value must be updated to the fresh evaluated data")
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

// TestKeys_ReturnsAllStoredKeys verifies that Keys returns every key currently held in the cache.
func TestKeys_ReturnsAllStoredKeys(t *testing.T) {
	c := newTestCache(t)
	c.Set("alpha", "1", 10*time.Second)
	c.Set("beta", "2", 10*time.Second)
	c.Set("gamma", "3", 10*time.Second)

	keys := c.Keys()
	assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, keys)
}

// --- MemoryMonitor tests ---

func TestMemoryMonitor_StartsWithNoPressure(t *testing.T) {
	// threshold near 100% — heap will never exceed it in a test
	m := newTestMonitor(t, "monitor-no-pressure", 0.99)
	pressure, pct := m.Status()
	assert.False(t, pressure)
	assert.GreaterOrEqual(t, pct, 0.0)
}

func TestMemoryMonitor_StopIsIdempotent(t *testing.T) {
	m := NewMemoryMonitor("monitor-idempotent", 0.99)
	m.Stop()
	assert.NotPanics(t, func() { m.Stop() }, "second Stop must not panic")
}

func TestMemoryMonitor_DetectsPressureAtZeroThreshold(t *testing.T) {
	// threshold 0.0 — any non-zero HeapInuse triggers pressure
	m := newTestMonitor(t, "monitor-zero-thresh", 0.0)
	time.Sleep(6 * time.Second) // wait for at least one tick
	pressure, _ := m.Status()
	assert.True(t, pressure, "zero threshold should always detect pressure after first tick")
}

// newTestMonitor creates a MemoryMonitor and registers Stop via t.Cleanup.
func newTestMonitor(t *testing.T, namespace string, maxMemPct float64) *MemoryMonitor {
	t.Helper()
	m := NewMemoryMonitor(namespace, maxMemPct)
	t.Cleanup(func() { m.Stop() })
	return m
}

// TestKeys_EmptyAfterClear verifies that Keys returns an empty slice once the cache is cleared.
func TestKeys_EmptyAfterClear(t *testing.T) {
	c := newTestCache(t)
	c.Set("x", "v", 10*time.Second)
	c.Clear()

	assert.Empty(t, c.Keys())
}

// TestStatus_DefaultsToNoPressure verifies Status returns false/0 before the memory monitor fires.
func TestStatus_DefaultsToNoPressure(t *testing.T) {
	c := newTestCache(t)
	pressure, pct := c.Status()
	assert.False(t, pressure)
	assert.Equal(t, 0.0, pct)
}

// TestStatus_ReflectsInternalState verifies Status returns the values set by monitorMemory.
func TestStatus_ReflectsInternalState(t *testing.T) {
	c := newTestCache(t)

	c.mu.Lock()
	c.isMemPressure = true
	c.lastUsedPct = 0.72
	c.mu.Unlock()

	pressure, pct := c.Status()
	assert.True(t, pressure)
	assert.InDelta(t, 0.72, pct, 0.0001)
}
