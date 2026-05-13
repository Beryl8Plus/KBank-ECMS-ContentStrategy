package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestMonitor creates a MemoryMonitor and registers Stop via t.Cleanup.
func newTestMonitor(t *testing.T, namespace string, maxMemPct float64) *MemoryMonitor {
	t.Helper()
	m := NewMemoryMonitor(namespace, maxMemPct)
	t.Cleanup(func() { m.Stop() })
	return m
}

// newTestCache creates a CacheMemory[string] with a high-threshold monitor
// (0.99) so memory pressure is never active during normal unit tests.
func newTestCache(t *testing.T) *CacheMemory[string] {
	t.Helper()
	m := newTestMonitor(t, t.Name()+"-monitor", 0.99)
	return NewCacheMemory[string](t.Name(), m, time.Hour)
}

// newPressuredCache creates a CacheMemory[string] backed by a zero-threshold
// monitor. After one tick (<=5 s) the monitor will flag pressure.
// Prime the cache with any keys that must exist BEFORE calling this, then
// sleep >=6s before the test assertion.
func newPressuredCache(t *testing.T, monitorNS string, cacheNS string) (*CacheMemory[string], *MemoryMonitor) {
	t.Helper()
	m := newTestMonitor(t, monitorNS, 0.0) // 0% threshold → always pressure after first tick
	c := NewCacheMemory[string](cacheNS, m, time.Hour)
	return c, m
}

// --- MemoryMonitor tests ---

func TestMemoryMonitor_StartsWithNoPressure(t *testing.T) {
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
	m := newTestMonitor(t, "monitor-zero-thresh", 0.0)
	time.Sleep(6 * time.Second)
	pressure, _ := m.Status()
	assert.True(t, pressure, "zero threshold should always detect pressure after first tick")
}

// --- CacheMemory tests ---

func TestGet_ReturnsHitBeforeExpiry(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "value", 10*time.Second)

	got, ok := c.Get("key")
	require.True(t, ok, "expected cache hit")
	assert.Equal(t, "value", got)
}

func TestGet_ReturnsMissAfterExpiry(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "value", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	got, ok := c.Get("key")
	assert.False(t, ok, "expected cache miss after TTL expiry")
	assert.Empty(t, got)
}

func TestGet_MissOnNonExistentKey(t *testing.T) {
	c := newTestCache(t)
	_, ok := c.Get("ghost")
	assert.False(t, ok)
}

// TestSet_RejectsNewEntryUnderMemPressure verifies that a brand-new key is not
// stored when the shared monitor reports memory pressure.
func TestSet_RejectsNewEntryUnderMemPressure(t *testing.T) {
	c, _ := newPressuredCache(t, "pressure-reject-monitor", "pressure-reject-cache")
	time.Sleep(6 * time.Second) // wait for monitor tick to activate pressure

	c.Set("key", "value", 10*time.Second)

	_, ok := c.Get("key")
	assert.False(t, ok, "new entry should not be stored under memory pressure")
}

// TestSet_UpdatesExistingEntryUnderMemPressure verifies that an already-cached
// key is refreshed even when memory pressure is active (net-zero memory change).
func TestSet_UpdatesExistingEntryUnderMemPressure(t *testing.T) {
	c, _ := newPressuredCache(t, "pressure-update-monitor", "pressure-update-cache")
	c.Set("key", "original", 10*time.Second) // prime before pressure activates

	time.Sleep(6 * time.Second) // wait for monitor tick to activate pressure

	c.Set("key", "refreshed", 10*time.Second)

	got, ok := c.Get("key")
	require.True(t, ok, "existing entry must be refreshable under memory pressure")
	assert.Equal(t, "refreshed", got)
}

func TestDelete_RemovesKey(t *testing.T) {
	c := newTestCache(t)
	c.Set("key", "value", 10*time.Second)
	c.Delete("key")

	_, ok := c.Get("key")
	assert.False(t, ok)
}

func TestClear_RemovesAllKeys(t *testing.T) {
	c := newTestCache(t)
	c.Set("a", "1", 10*time.Second)
	c.Set("b", "2", 10*time.Second)
	c.Clear()

	assert.Equal(t, 0, c.cache.ItemCount(), "cache should be empty after Clear")
}
