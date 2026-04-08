package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kbank-ecms/internal/domain/entity"
)

// mockScheduleRepository is a local in-package mock for domainrepo.ScheduleRepository.
// Function fields default to nil — set only what the specific test needs.
type mockScheduleRepository struct {
	checkFn         func(ctx context.Context, decisionRuleID uuid.UUID, placementID uuid.UUID, effectiveFrom, effectiveUntil time.Time, excludeID *uuid.UUID) (*entity.Schedule, error)
	createFn        func(ctx context.Context, schedule *entity.Schedule) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)
	listFn          func(ctx context.Context) ([]*entity.Schedule, error)
	listPaginatedFn func(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error)
	updateFn        func(ctx context.Context, schedule *entity.Schedule) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
}

func (m *mockScheduleRepository) CheckScheduleOverlap(
	ctx context.Context,
	decisionRuleID uuid.UUID,
	placementID uuid.UUID,
	effectiveFrom, effectiveUntil time.Time,
	excludeID *uuid.UUID,
) (*entity.Schedule, error) {
	return m.checkFn(ctx, decisionRuleID, placementID, effectiveFrom, effectiveUntil, excludeID)
}

func (m *mockScheduleRepository) CreateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if m.createFn != nil {
		return m.createFn(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockScheduleRepository) ListSchedules(ctx context.Context) ([]*entity.Schedule, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockScheduleRepository) ListSchedulesPaginated(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error) {
	if m.listPaginatedFn != nil {
		return m.listPaginatedFn(ctx, page, limit)
	}
	return nil, 0, nil
}

func (m *mockScheduleRepository) UpdateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func TestScheduleService_ValidateScheduleOverlap(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	placementID := uuid.New()
	existingID := uuid.New()

	// conflictSchedule simulates an existing schedule returned by the repository.
	conflictSchedule := &entity.Schedule{
		EffectiveFrom:  now,
		EffectiveUntil: later,
	}
	conflictSchedule.ID = existingID
	conflictSchedule.PlacementID = placementID

	tests := []struct {
		name           string
		schedule       *entity.Schedule
		repoReturn     *entity.Schedule
		repoErr        error
		wantErr        bool
		errContains    string
		checkExcludeID func(t *testing.T, got *uuid.UUID)
	}{
		{
			name:       "no overlap found - returns nil error",
			schedule:   &entity.Schedule{PlacementID: placementID, EffectiveFrom: now, EffectiveUntil: later},
			repoReturn: nil,
			wantErr:    false,
		},
		{
			name:        "overlap detected - returns descriptive error",
			schedule:    &entity.Schedule{PlacementID: placementID, EffectiveFrom: now, EffectiveUntil: later},
			repoReturn:  conflictSchedule,
			wantErr:     true,
			errContains: "overlaps with existing schedule",
		},
		{
			name:        "error contains placement ID and date range",
			schedule:    &entity.Schedule{PlacementID: placementID, EffectiveFrom: now, EffectiveUntil: later},
			repoReturn:  conflictSchedule,
			wantErr:     true,
			errContains: placementID.String(),
		},
		{
			name:        "repo error is wrapped and propagated",
			schedule:    &entity.Schedule{PlacementID: placementID, EffectiveFrom: now, EffectiveUntil: later},
			repoErr:     errors.New("connection refused"),
			wantErr:     true,
			errContains: "validating schedule overlap",
		},
		{
			name: "new schedule (zero ID) passes nil excludeID",
			schedule: &entity.Schedule{
				// ID is zero-value uuid.UUID{}
				PlacementID:    placementID,
				EffectiveFrom:  now,
				EffectiveUntil: later,
			},
			repoReturn: nil,
			wantErr:    false,
			checkExcludeID: func(t *testing.T, got *uuid.UUID) {
				t.Helper()
				assert.Nil(t, got, "new schedule should pass nil excludeID to repo")
			},
		},
		{
			name: "self-update: existing schedule ID is excluded from check",
			schedule: func() *entity.Schedule {
				s := &entity.Schedule{
					PlacementID:    placementID,
					EffectiveFrom:  now,
					EffectiveUntil: later,
				}
				s.ID = existingID
				return s
			}(),
			repoReturn: nil,
			wantErr:    false,
			checkExcludeID: func(t *testing.T, got *uuid.UUID) {
				t.Helper()
				require.NotNil(t, got, "update should pass non-nil excludeID to repo")
				assert.Equal(t, existingID, *got)
			},
		},
		{
			name: "adjacent ranges (same date boundary) - no overlap",
			schedule: &entity.Schedule{
				PlacementID:    placementID,
				EffectiveFrom:  later,                          // starts exactly when 'now-later' ends
				EffectiveUntil: later.Add(24 * 30 * time.Hour), // 30 days later
			},
			repoReturn: nil,
			wantErr:    false,
		},
		{
			name: "different placement ID - no overlap check conflict",
			schedule: &entity.Schedule{
				PlacementID:    uuid.New(), // different placement
				EffectiveFrom:  now,
				EffectiveUntil: later,
			},
			repoReturn: nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var capturedExcludeID *uuid.UUID
			mock := &mockScheduleRepository{
				checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, excludeID *uuid.UUID) (*entity.Schedule, error) {
					capturedExcludeID = excludeID
					return tt.repoReturn, tt.repoErr
				},
			}
			svc := NewScheduleService(mock)
			err := svc.ValidateScheduleOverlap(context.Background(), tt.schedule)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			if tt.checkExcludeID != nil {
				tt.checkExcludeID(t, capturedExcludeID)
			}
		})
	}
}

func TestScheduleService_CreateSchedule(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	placementID := uuid.New()
	schedule := &entity.Schedule{PlacementID: placementID, EffectiveFrom: now, EffectiveUntil: later}

	t.Run("no overlap - creates successfully", func(t *testing.T) {
		t.Parallel()
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
		}
		svc := NewScheduleService(mock)
		err := svc.CreateSchedule(context.Background(), schedule)
		require.NoError(t, err)
	})

	t.Run("overlap found - returns error without calling create", func(t *testing.T) {
		t.Parallel()
		conflict := &entity.Schedule{EffectiveFrom: now, EffectiveUntil: later}
		conflict.PlacementID = placementID
		createCalled := false
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return conflict, nil
			},
			createFn: func(_ context.Context, _ *entity.Schedule) error {
				createCalled = true
				return nil
			},
		}
		svc := NewScheduleService(mock)
		err := svc.CreateSchedule(context.Background(), schedule)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overlaps")
		assert.False(t, createCalled, "create should not be called when overlap is found")
	})

	t.Run("repo create error is propagated", func(t *testing.T) {
		t.Parallel()
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
			createFn: func(_ context.Context, _ *entity.Schedule) error {
				return errors.New("db connection lost")
			},
		}
		svc := NewScheduleService(mock)
		err := svc.CreateSchedule(context.Background(), schedule)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db connection lost")
	})
}

func TestScheduleService_GetScheduleByID(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	found := &entity.Schedule{}
	found.ID = id

	tests := []struct {
		name       string
		repoReturn *entity.Schedule
		repoErr    error
		wantNil    bool
		wantErr    bool
	}{
		{"found", found, nil, false, false},
		{"not found returns nil", nil, nil, true, false},
		{"repo error propagated", nil, errors.New("timeout"), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := &mockScheduleRepository{
				checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
					return nil, nil
				},
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) { return tt.repoReturn, tt.repoErr },
			}
			svc := NewScheduleService(mock)
			result, err := svc.GetScheduleByID(context.Background(), id)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.wantNil {
					assert.Nil(t, result)
				} else {
					assert.Equal(t, tt.repoReturn, result)
				}
			}
		})
	}
}

func TestScheduleService_ListSchedules(t *testing.T) {
	t.Parallel()

	s1, s2 := &entity.Schedule{}, &entity.Schedule{}
	s1.ID, s2.ID = uuid.New(), uuid.New()

	t.Run("returns list from repo", func(t *testing.T) {
		t.Parallel()
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
			listFn: func(_ context.Context) ([]*entity.Schedule, error) { return []*entity.Schedule{s1, s2}, nil },
		}
		svc := NewScheduleService(mock)
		result, err := svc.ListSchedules(context.Background())
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		t.Parallel()
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
			listFn: func(_ context.Context) ([]*entity.Schedule, error) { return nil, errors.New("db down") },
		}
		svc := NewScheduleService(mock)
		result, err := svc.ListSchedules(context.Background())
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestScheduleService_UpdateSchedule(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	existing := &entity.Schedule{PlacementID: uuid.New(), EffectiveFrom: now, EffectiveUntil: later}
	existing.ID = uuid.New()

	t.Run("no overlap - updates successfully", func(t *testing.T) {
		t.Parallel()
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
			updateFn: func(_ context.Context, _ *entity.Schedule) error { return nil },
		}
		svc := NewScheduleService(mock)
		require.NoError(t, svc.UpdateSchedule(context.Background(), existing))
	})

	t.Run("overlap found returns error without calling update", func(t *testing.T) {
		t.Parallel()
		conflict := &entity.Schedule{EffectiveFrom: now, EffectiveUntil: later}
		conflict.PlacementID = existing.PlacementID
		updateCalled := false
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return conflict, nil
			},
			updateFn: func(_ context.Context, _ *entity.Schedule) error { updateCalled = true; return nil },
		}
		svc := NewScheduleService(mock)
		err := svc.UpdateSchedule(context.Background(), existing)
		require.Error(t, err)
		assert.False(t, updateCalled)
	})
}

func TestScheduleService_DeleteSchedule(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	t.Run("delegates to repo", func(t *testing.T) {
		t.Parallel()
		called := false
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
			deleteFn: func(_ context.Context, got uuid.UUID) error { called = true; assert.Equal(t, id, got); return nil },
		}
		svc := NewScheduleService(mock)
		require.NoError(t, svc.DeleteSchedule(context.Background(), id))
		assert.True(t, called)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		t.Parallel()
		mock := &mockScheduleRepository{
			checkFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
				return nil, nil
			},
			deleteFn: func(_ context.Context, _ uuid.UUID) error { return errors.New("fk constraint") },
		}
		svc := NewScheduleService(mock)
		require.Error(t, svc.DeleteSchedule(context.Background(), id))
	})
}
