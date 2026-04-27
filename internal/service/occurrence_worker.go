package service

import (
	"context"
	"fmt"
	"time"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

// OccurrenceWorkerConfig holds timing parameters for the background worker.
type OccurrenceWorkerConfig struct {
	// MaterializationInterval is how often the worker materialises occurrences
	// for the rolling window. Defaults to 1 hour when zero.
	MaterializationInterval time.Duration

	// ExpiryInterval is how often the worker expires ended occurrences.
	// Defaults to 15 minutes when zero.
	ExpiryInterval time.Duration
}

func (c OccurrenceWorkerConfig) materializationInterval() time.Duration {
	if c.MaterializationInterval <= 0 {
		return time.Hour
	}
	return c.MaterializationInterval
}

func (c OccurrenceWorkerConfig) expiryInterval() time.Duration {
	if c.ExpiryInterval <= 0 {
		return 15 * time.Minute
	}
	return c.ExpiryInterval
}

// durationUntilMidnight returns the duration from now until the next 00:00:00
// in the local timezone.
func durationUntilMidnight(timezone *time.Location) time.Duration {
	now := time.Now().In(timezone)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, timezone)
	return time.Until(next)
}

// OccurrenceWorker runs the materialization and cleanup jobs on configurable
// intervals. It is safe for concurrent use: a single goroutine owns each
// ticker so there are no shared-state races.
//
// Usage:
//
//	worker := NewOccurrenceWorker(svc, cfg)
//	go worker.Start(ctx)
//	// cancel ctx to stop gracefully
type OccurrenceWorker struct {
	svc *ScheduleMaterializationService
	cfg OccurrenceWorkerConfig
}

// NewOccurrenceWorker creates a new OccurrenceWorker.
func NewOccurrenceWorker(svc *ScheduleMaterializationService, cfg OccurrenceWorkerConfig) *OccurrenceWorker {
	return &OccurrenceWorker{svc: svc, cfg: cfg}
}

// Start blocks until ctx is cancelled, running two tickers:
//  1. A materialisation ticker that calls MaterializeWindow at every
//     MaterializationInterval. The first materialisation is run immediately
//     on start (before the first tick) so the table is populated as soon as
//     the service comes up.
//  2. A cleanup ticker that calls CleanupPastOccurrences at every
//     CleanupInterval.
//
// Both operations log errors and continue — a transient DB failure does not
// stop the worker.
func (w *OccurrenceWorker) Start(ctx context.Context) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "OCCURRENCE-WORKER",
		Level:   "INFO",
		Message: fmt.Sprintf("occurrence worker starting (materialize_interval=%s cleanup_at=00:00 expiry_interval=%s)",
			w.cfg.materializationInterval(), w.cfg.expiryInterval()),
	})

	// Run once immediately so occurrences are ready on service startup.
	w.runMaterialization(ctx)

	// Asia/Bangkok is UTC+7, so the cleanup runs at 17:00 UTC, which is a low-traffic time for the system.
	timezone, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "OCCURRENCE-WORKER",
			Level:   "ERROR",
			Message: fmt.Sprintf("failed to load timezone: %v", err),
		})
	}
	materializeTicker := time.NewTicker(w.cfg.materializationInterval())
	cleanupTimer := time.NewTimer(durationUntilMidnight(timezone))
	expiryTicker := time.NewTicker(w.cfg.expiryInterval())
	defer materializeTicker.Stop()
	defer cleanupTimer.Stop()
	defer expiryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.LSystem(ctx, entity.SystemLog{Service: "OCCURRENCE-WORKER", Level: "INFO", Message: "occurrence worker stopped"})
			return
		case <-materializeTicker.C:
			w.runMaterialization(ctx)
		case <-cleanupTimer.C:
			w.runCleanup(ctx)
			cleanupTimer.Reset(durationUntilMidnight(timezone))
		case <-expiryTicker.C:
			w.runExpiry(ctx)
		}
	}
}

func (w *OccurrenceWorker) runMaterialization(ctx context.Context) {
	if err := w.svc.MaterializeWindow(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "OCCURRENCE-WORKER", Level: "ERROR", Message: "occurrence materialization failed: " + err.Error()})
	}
}

func (w *OccurrenceWorker) runCleanup(ctx context.Context) {
	if err := w.svc.CleanupPastOccurrences(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "OCCURRENCE-WORKER", Level: "ERROR", Message: "occurrence cleanup failed: " + err.Error()})
	}
}

func (w *OccurrenceWorker) runExpiry(ctx context.Context) {
	if err := w.svc.ExpireEndedOccurrences(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "OCCURRENCE-WORKER", Level: "ERROR", Message: "occurrence expiry failed: " + err.Error()})
	}
}
