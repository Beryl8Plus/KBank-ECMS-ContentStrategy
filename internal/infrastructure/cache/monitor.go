package cache

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

// MemoryMonitor runs a single background goroutine that reads MemStats once per
// tick and tracks whether heap usage has exceeded the configured threshold.
//
// All CacheMemory instances that share this monitor query it for pressure state,
// reducing stop-the-world runtime.ReadMemStats calls from N (one per cache) to 1.
type MemoryMonitor struct {
	maxMemoryPct float64
	stopCh       chan struct{}
	once         sync.Once // ensures Stop is idempotent

	mu              sync.RWMutex
	isUnderPressure bool
	lastUsedPct     float64

	mMemPressureGauge prometheus.Gauge
}

// NewMemoryMonitor initialises a shared memory monitor and starts its background ticker.
// namespace is used as a Prometheus metric prefix for the memory_pressure_active gauge.
// maxMemPct is the HeapInuse/HeapSys ratio (0–1) above which pressure is flagged.
func NewMemoryMonitor(namespace string, maxMemPct float64) *MemoryMonitor {
	pressureGauge := mustRegister(prometheus.NewGauge(prometheus.GaugeOpts{
		Name: namespace + "_memory_pressure_active",
		Help: "Indicates if the memory pressure threshold is active (1 = active, 0 = inactive)",
	}))

	mm := &MemoryMonitor{
		maxMemoryPct:      maxMemPct,
		stopCh:            make(chan struct{}),
		mMemPressureGauge: pressureGauge,
	}
	go mm.run()
	return mm
}

// IsUnderPressure returns true when the last measured heap usage exceeded the threshold.
func (mm *MemoryMonitor) IsUnderPressure() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.isUnderPressure
}

// Status returns the current pressure flag and the last measured heap utilisation
// ratio (HeapInuse/HeapSys, range 0–1). Returns false/0 before the first tick.
func (mm *MemoryMonitor) Status() (isUnderPressure bool, usedPct float64) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.isUnderPressure, mm.lastUsedPct
}

// Stop shuts down the background monitor goroutine. Safe to call multiple times.
func (mm *MemoryMonitor) Stop() {
	mm.once.Do(func() { close(mm.stopCh) })
}

func (mm *MemoryMonitor) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var m runtime.MemStats

	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&m)

			if m.HeapSys == 0 {
				continue
			}

			usedPct := float64(m.HeapInuse) / float64(m.HeapSys)

			mm.mu.Lock()
			mm.lastUsedPct = usedPct
			if usedPct > mm.maxMemoryPct {
				mm.isUnderPressure = true
				mm.mMemPressureGauge.Set(1)
			} else {
				mm.isUnderPressure = false
				mm.mMemPressureGauge.Set(0)
			}
			mm.mu.Unlock()

			if usedPct > 0.8 {
				logger.LSystem(context.Background(), entity.SystemLog{
					Service: "Memory",
					Level:   "WARN",
					Message: fmt.Sprintf("critical memory pressure (%.1f%% heap in use) — triggering GC", usedPct*100),
				})
				runtime.GC()
			}

		case <-mm.stopCh:
			return
		}
	}
}
