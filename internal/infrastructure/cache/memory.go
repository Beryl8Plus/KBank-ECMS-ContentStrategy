package cache

import (
	"errors"
	"runtime"
	"sync"
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
// It monitors heap usage and evicts entries under memory pressure.
// Backed by github.com/patrickmn/go-cache for TTL management and eviction.
type CacheMemory[T any] struct {
	cache *gocache.Cache

	// Config
	maxMemoryPct float64 // e.g. 0.60 (60%)

	// Lifecycle
	stopCh chan struct{}

	// Status
	mu            sync.RWMutex
	isMemPressure bool

	// Metrics (per-instance, namespaced)
	mCacheHits        prometheus.Counter
	mCacheMisses      prometheus.Counter
	mMemPressureGauge prometheus.Gauge
	mCacheSizeGauge   prometheus.Gauge
}

// NewCacheMemory initializes the cache and starts the background memory monitor.
// namespace is used as a prefix for Prometheus metric names and must be unique per instance.
func NewCacheMemory[T any](namespace string, maxMemPct float64, ttl time.Duration) *CacheMemory[T] {
	cacheHits := mustRegister(prometheus.NewCounter(prometheus.CounterOpts{
		Name: namespace + "_cache_hits_total",
		Help: "Total number of cache hits in local memory",
	}))

	cacheMisses := mustRegister(prometheus.NewCounter(prometheus.CounterOpts{
		Name: namespace + "_cache_misses_total",
		Help: "Total number of cache misses in local memory",
	}))

	memPressure := mustRegister(prometheus.NewGauge(prometheus.GaugeOpts{
		Name: namespace + "_memory_pressure_active",
		Help: "Indicates if the memory pressure threshold is active (1 = active, 0 = inactive)",
	}))

	cacheSize := mustRegister(prometheus.NewGauge(prometheus.GaugeOpts{
		Name: namespace + "_cache_entries_count",
		Help: "Current number of entries in the local cache",
	}))

	rc := &CacheMemory[T]{
		cache:             gocache.New(ttl, 5*time.Minute),
		maxMemoryPct:      maxMemPct,
		stopCh:            make(chan struct{}),
		mCacheHits:        cacheHits,
		mCacheMisses:      cacheMisses,
		mMemPressureGauge: memPressure,
		mCacheSizeGauge:   cacheSize,
	}

	go rc.monitorMemory()
	return rc
}

// Stop shuts down the background memory monitor goroutine.
func (rc *CacheMemory[T]) Stop() {
	close(rc.stopCh)
}

// Get retrieves a value from local memory.
// Returns the zero value and false when the key is absent or expired.
// The context is accepted for future tracing/cancellation support.
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

// Set adds a value with the given TTL. Under memory pressure new entries are
// rejected and the key is evicted if it already exists.
func (rc *CacheMemory[T]) Set(key string, value T, ttl time.Duration) {
	rc.mu.RLock()
	pressure := rc.isMemPressure
	rc.mu.RUnlock()

	if pressure {
		rc.cache.Delete(key)
		rc.updateMetrics()
		return
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

// monitorMemory runs in the background to check HeapAlloc against the threshold.
func (rc *CacheMemory[T]) monitorMemory() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var m runtime.MemStats

	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&m)

			// Guard: HeapSys can be zero immediately after startup.
			if m.HeapSys == 0 {
				continue
			}

			// Always purge expired entries on every tick, regardless of memory pressure.
			rc.purgeExpired()

			// Use HeapAlloc/HeapSys — a meaningful heap-specific utilisation ratio.
			usedPct := float64(m.HeapAlloc) / float64(m.HeapSys)

			rc.mu.Lock()
			if usedPct > rc.maxMemoryPct {
				rc.isMemPressure = true
				rc.mMemPressureGauge.Set(1)
			} else {
				rc.isMemPressure = false
				rc.mMemPressureGauge.Set(0)
			}
			rc.mu.Unlock()

			// Critical pressure: wipe all remaining entries.
			if usedPct > 0.8 {
				rc.cache.Flush()
			}

			rc.updateMetrics()

		case <-rc.stopCh:
			return
		}
	}
}

// updateMetrics updates the Prometheus gauge for cache size.
func (rc *CacheMemory[T]) updateMetrics() {
	rc.mCacheSizeGauge.Set(float64(rc.cache.ItemCount()))
}

// purgeExpired removes entries that are past their TTL.
func (rc *CacheMemory[T]) purgeExpired() {
	rc.cache.DeleteExpired()
}
