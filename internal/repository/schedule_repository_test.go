package repository

import (
	"testing"

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
