package service

import (
	"context"
	"fmt"
	"time"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

// AttributeSyncWorkerConfig holds timing parameters for the background worker.
type AttributeSyncWorkerConfig struct {
	// SyncHour and SyncMinute define the daily local time to run attribute sync.
	// Defaults to 03:00 when both are zero.
	SyncHour   int
	SyncMinute int

	// IntegrityInterval controls how often the integrity checker runs.
	// Defaults to 5 minutes when zero.
	IntegrityInterval time.Duration
}

func (c AttributeSyncWorkerConfig) integrityInterval() time.Duration {
	if c.IntegrityInterval <= 0 {
		return 5 * time.Minute
	}
	return c.IntegrityInterval
}

// nextDailyRun returns the duration until the next occurrence of hour:minute (local time).
func nextDailyRun(hour, minute int) time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return time.Until(next)
}

// AttributeSyncWorker runs the attribute sync and integrity checker on
// configurable intervals. It mirrors the OccurrenceWorker pattern.
//
// Usage:
//
//	worker := NewAttributeSyncWorker(syncSvc, checkerSvc, cfg)
//	go worker.Start(ctx)
type AttributeSyncWorker struct {
	syncSvc    *AttributeSyncService
	checkerSvc *IntegrityCheckerService
	cfg        AttributeSyncWorkerConfig
}

// NewAttributeSyncWorker creates a new AttributeSyncWorker.
func NewAttributeSyncWorker(
	syncSvc *AttributeSyncService,
	checkerSvc *IntegrityCheckerService,
	cfg AttributeSyncWorkerConfig,
) *AttributeSyncWorker {
	return &AttributeSyncWorker{syncSvc: syncSvc, checkerSvc: checkerSvc, cfg: cfg}
}

// Start blocks until ctx is cancelled. On startup it immediately runs the
// integrity checker (sync is deferred to the next scheduled 03:00 run to avoid
// blocking startup when the external API is slow).
func (w *AttributeSyncWorker) Start(ctx context.Context) {
	syncHour := w.cfg.SyncHour
	syncMinute := w.cfg.SyncMinute
	if syncHour == 0 && syncMinute == 0 {
		syncHour = 3
	}

	nextRun := nextDailyRun(syncHour, syncMinute)
	logger.LSystem(ctx, entity.SystemLog{
		Service: "ATTRIBUTE-SYNC-WORKER",
		Level:   "INFO",
		Message: fmt.Sprintf("worker starting (daily_sync=%02d:%02d next_in=%s integrity_interval=%s)",
			syncHour, syncMinute, nextRun.Round(time.Second), w.cfg.integrityInterval()),
	})

	// Run integrity check immediately on startup.
	w.runIntegrityCheck(ctx)

	syncTimer := time.NewTimer(nextRun)
	integrityTicker := time.NewTicker(w.cfg.integrityInterval())
	defer syncTimer.Stop()
	defer integrityTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.LSystem(ctx, entity.SystemLog{
				Service: "ATTRIBUTE-SYNC-WORKER",
				Level:   "INFO",
				Message: "worker stopped",
			})
			return
		case <-syncTimer.C:
			w.runSync(ctx)
			syncTimer.Reset(nextDailyRun(syncHour, syncMinute))
		case <-integrityTicker.C:
			w.runIntegrityCheck(ctx)
		}
	}
}

func (w *AttributeSyncWorker) runSync(ctx context.Context) {
	if err := w.syncSvc.Sync(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "ATTRIBUTE-SYNC-WORKER",
			Level:   "ERROR",
			Message: "attribute sync failed: " + err.Error(),
		})
	}
}

func (w *AttributeSyncWorker) runIntegrityCheck(ctx context.Context) {
	if err := w.checkerSvc.RunCheck(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "ATTRIBUTE-SYNC-WORKER",
			Level:   "ERROR",
			Message: "integrity check failed: " + err.Error(),
		})
	}
}
