package cache

import (
	"errors"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

// mustRegister registers c in the default Prometheus registry.
// If a collector with the same name is already registered, the existing
// collector is reused — making it safe to call from parallel tests or
// when multiple CacheMemory instances share the same namespace.
func mustRegister[C prometheus.Collector](c C) C {
	if err := prometheus.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return are.ExistingCollector.(C)
		}
		panic(err)
	}
	return c
}

// CacheMemory is a memory-aware in-process cache for any value type T.
// Each entry carries its own TTL supplied at Set time.
// Memory pressure is delegated to a shared MemoryMonitor — no per-instance
// goroutine or ReadMemStats call.
// Backed by github.com/patrickmn/go-cache for TTL management and eviction.
type CacheMemory[T any] struct {
	cache   *gocache.Cache
	monitor *MemoryMonitor

	// Metrics (per-instance, namespaced)
	mCacheHits      prometheus.Counter
	mCacheMisses    prometheus.Counter
	mCacheSizeGauge prometheus.Gauge
}

// NewCacheMemory initialises the cache.
// namespace is used as a prefix for per-instance Prometheus metric names; must be unique.
// monitor is the shared MemoryMonitor that tracks heap pressure for this cache.
// ttl sets the default item lifetime and the underlying go-cache cleanup interval baseline.
func NewCacheMemory[T any](namespace string, monitor *MemoryMonitor, ttl time.Duration) *CacheMemory[T] {
	cacheHits := mustRegister(prometheus.NewCounter(prometheus.CounterOpts{
		Name: namespace + "_cache_hits_total",
		Help: "Total number of cache hits in local memory",
	}))

	cacheMisses := mustRegister(prometheus.NewCounter(prometheus.CounterOpts{
		Name: namespace + "_cache_misses_total",
		Help: "Total number of cache misses in local memory",
	}))

	cacheSize := mustRegister(prometheus.NewGauge(prometheus.GaugeOpts{
		Name: namespace + "_cache_entries_count",
		Help: "Current number of entries in the local cache",
	}))

	return &CacheMemory[T]{
		cache:           gocache.New(ttl, 5*time.Second),
		monitor:         monitor,
		mCacheHits:      cacheHits,
		mCacheMisses:    cacheMisses,
		mCacheSizeGauge: cacheSize,
	}
}

// Stop is a no-op. Lifecycle is managed by MemoryMonitor.Stop().
// Kept for call-site backward compatibility.
func (rc *CacheMemory[T]) Stop() {}

// Get retrieves a value from local memory.
// Returns the zero value and false when the key is absent or expired.
func (rc *CacheMemory[T]) Get(key string) (T, bool) {
	v, found := rc.cache.Get(key)
	if !found {
		rc.mCacheMisses.Inc()
		var zero T
		return zero, false
	}
	rc.mCacheHits.Inc()
	return v.(T), true
}

// Set adds or replaces a value with the given TTL.
// Under memory pressure, existing keys are still updated (stale data replaced
// with fresh evaluated data is net-zero on memory). Only brand-new keys are
// rejected to avoid growing the memory footprint.
func (rc *CacheMemory[T]) Set(key string, value T, ttl time.Duration) {
	if rc.monitor.IsUnderPressure() {
		if _, exists := rc.cache.Get(key); !exists {
			rc.updateMetrics()
			return
		}
	}
	rc.cache.Set(key, value, ttl)
	rc.updateMetrics()
}

// Delete removes a single key from local memory.
func (rc *CacheMemory[T]) Delete(key string) {
	rc.cache.Delete(key)
	rc.updateMetrics()
}

// Clear removes all keys from local memory.
func (rc *CacheMemory[T]) Clear() {
	rc.cache.Flush()
	rc.updateMetrics()
}

// Keys returns a slice of all keys currently stored in the cache.
func (rc *CacheMemory[T]) Keys() []string {
	keys := make([]string, 0, rc.cache.ItemCount())
	for k := range rc.cache.Items() {
		keys = append(keys, k)
	}
	return keys
}

// Status returns whether the cache is under memory pressure and the last
// measured heap utilisation ratio from the shared monitor (range 0–1).
// Returns false/0 before the monitor's first tick fires.
func (rc *CacheMemory[T]) Status() (isMemPressure bool, usedPct float64) {
	return rc.monitor.Status()
}

// updateMetrics updates the Prometheus gauge for cache size.
func (rc *CacheMemory[T]) updateMetrics() {
	rc.mCacheSizeGauge.Set(float64(rc.cache.ItemCount()))
}
