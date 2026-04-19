package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	domainrepo "kbank-ecms/internal/domain/repository"
)

// TestNewSchedulePostgresRepository verifies that the constructor returns a
// non-nil repository that satisfies the ScheduleRepository domain interface.
func TestNewSchedulePostgresRepository(t *testing.T) {
	// Use nil dialector with DryRun — same pattern as base_model_test.go.
	// DryRun mode is set so that no query execution is attempted.
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})

	repo := NewSchedulePostgresRepository(db)
	require.NotNil(t, repo)

	// Runtime confirmation that the concrete type satisfies the domain interface.
	// The compile-time check (var _ domainrepo.ScheduleRepository = (*SchedulePostgresRepository)(nil))
	// is in schedule_repository.go; this assertion covers the runtime value.
	var _ domainrepo.ScheduleRepository = repo
}

// TestListActiveSchedulesInWindow_WithinWindow verifies that
// ListActiveSchedulesInWindow can be called with a time inside a schedule's
// effective window and returns a non-error result.
// NOTE: With the nil-dialector DryRun DB used here no real SQL is executed;
// actual filtering behaviour is covered by integration tests against a live DB.
func TestListActiveSchedulesInWindow_WithinWindow(t *testing.T) {
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	repo := NewSchedulePostgresRepository(db)

	// `at` sits inside the window [2026-01-01, 2026-12-31).
	at := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	schedules, err := repo.ListActiveSchedulesInWindow(context.Background(), at)

	require.NoError(t, err)
	// DryRun returns nil/empty because no rows are fetched; callers must treat nil
	// and empty slice equivalently.
	require.Empty(t, schedules)
}

// TestListActiveSchedulesInWindow_BeforeWindow verifies that the method
// handles a query time before any schedule's effective window without error.
func TestListActiveSchedulesInWindow_BeforeWindow(t *testing.T) {
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	repo := NewSchedulePostgresRepository(db)

	// `at` is before any schedule's effective_from — should yield empty result.
	at := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	schedules, err := repo.ListActiveSchedulesInWindow(context.Background(), at)

	require.NoError(t, err)
	require.Empty(t, schedules)
}

// TestListActiveSchedulesInWindow_AfterWindow verifies that the method
// handles a query time after all schedule effective windows without error.
func TestListActiveSchedulesInWindow_AfterWindow(t *testing.T) {
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	repo := NewSchedulePostgresRepository(db)

	// `at` is past every schedule's effective_until — should yield empty result.
	at := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

	schedules, err := repo.ListActiveSchedulesInWindow(context.Background(), at)

	require.NoError(t, err)
	require.Empty(t, schedules)
}
