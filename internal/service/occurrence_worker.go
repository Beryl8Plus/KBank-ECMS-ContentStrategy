package service

import (
	"context"
	"log/slog"
	"time"
)

// OccurrenceWorkerConfig holds timing parameters for the background worker.
type OccurrenceWorkerConfig struct {
	// MaterializationInterval is how often the worker materialises occurrences
	// for the rolling window. Defaults to 1 hour when zero.
	MaterializationInterval time.Duration

	// CleanupInterval is how often the worker runs the cleanup job.
	// Defaults to 24 hours when zero.
	CleanupInterval time.Duration
}

func (c OccurrenceWorkerConfig) materializationInterval() time.Duration {
	if c.MaterializationInterval <= 0 {
		return time.Hour
	}
	return c.MaterializationInterval
}

func (c OccurrenceWorkerConfig) cleanupInterval() time.Duration {
	if c.CleanupInterval <= 0 {
		return 24 * time.Hour
	}
	return c.CleanupInterval
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
	slog.InfoContext(ctx, "occurrence worker starting",
		slog.Duration("materialize_interval", w.cfg.materializationInterval()),
		slog.Duration("cleanup_interval", w.cfg.cleanupInterval()),
	)

	// Run once immediately so occurrences are ready on service startup.
	w.runMaterialization(ctx)

	materializeTicker := time.NewTicker(w.cfg.materializationInterval())
	cleanupTicker := time.NewTicker(w.cfg.cleanupInterval())
	defer materializeTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "occurrence worker stopped")
			return
		case <-materializeTicker.C:
			w.runMaterialization(ctx)
		case <-cleanupTicker.C:
			w.runCleanup(ctx)
		}
	}
}

func (w *OccurrenceWorker) runMaterialization(ctx context.Context) {
	if err := w.svc.MaterializeWindow(ctx); err != nil {
		slog.ErrorContext(ctx, "occurrence materialization failed", slog.String("error", err.Error()))
	}
}

func (w *OccurrenceWorker) runCleanup(ctx context.Context) {
	if err := w.svc.CleanupPastOccurrences(ctx); err != nil {
		slog.ErrorContext(ctx, "occurrence cleanup failed", slog.String("error", err.Error()))
	}
}
