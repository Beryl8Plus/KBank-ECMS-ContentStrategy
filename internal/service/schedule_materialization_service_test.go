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
	"kbank-ecms/internal/domain/entity/enums"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockOccurrenceRepository implements domainrepo.ScheduleOccurrenceRepository
// for unit tests. All function fields default to nil (no-op or zero-return).
type mockOccurrenceRepository struct {
	upsertFn           func(ctx context.Context, occurrences []*entity.ScheduleOccurrence) error
	deleteFutureFn     func(ctx context.Context, scheduleID uuid.UUID, after time.Time) error
	deletePastFn       func(ctx context.Context, before time.Time) error
	listActiveAtFn     func(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error)
	listByScheduleIDFn func(ctx context.Context, scheduleID uuid.UUID, page, limit int) ([]*entity.ScheduleOccurrence, int64, error)
}

func (m *mockOccurrenceRepository) UpsertOccurrences(ctx context.Context, occurrences []*entity.ScheduleOccurrence) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, occurrences)
	}
	return nil
}

func (m *mockOccurrenceRepository) DeleteFutureByScheduleID(ctx context.Context, scheduleID uuid.UUID, after time.Time) error {
	if m.deleteFutureFn != nil {
		return m.deleteFutureFn(ctx, scheduleID, after)
	}
	return nil
}

func (m *mockOccurrenceRepository) DeletePastOccurrences(ctx context.Context, before time.Time) error {
	if m.deletePastFn != nil {
		return m.deletePastFn(ctx, before)
	}
	return nil
}

func (m *mockOccurrenceRepository) ListActiveAt(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error) {
	if m.listActiveAtFn != nil {
		return m.listActiveAtFn(ctx, at)
	}
	return nil, nil
}

func (m *mockOccurrenceRepository) ListActiveByPlacementsAt(ctx context.Context, placementNames []string, at time.Time) ([]*entity.ScheduleOccurrence, error) {
	return nil, nil
}

func (m *mockOccurrenceRepository) ListByScheduleID(ctx context.Context, scheduleID uuid.UUID, page, limit int) ([]*entity.ScheduleOccurrence, int64, error) {
	if m.listByScheduleIDFn != nil {
		return m.listByScheduleIDFn(ctx, scheduleID, page, limit)
	}
	return nil, 0, nil
}

// ---------------------------------------------------------------------------
// Helper: build a schedule with a given recurrence type
// ---------------------------------------------------------------------------

func makeSchedule(recurrenceType enums.RecurrenceType, rule *string, allDay bool, todStart, todEnd *string) *entity.Schedule {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := &entity.Schedule{
		RecurrenceType: recurrenceType,
		RecurrenceRule: rule,
		EffectiveFrom:  now,
		EffectiveUntil: now.Add(30 * 24 * time.Hour), // 30 days
		AllDay:         allDay,
		TimeOfDayStart: todStart,
		TimeOfDayEnd:   todEnd,
		Timezone:       "UTC",
		IsActive:       true,
	}
	s.ID = uuid.New()
	return s
}

func ptr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// generateOccurrences unit tests
// ---------------------------------------------------------------------------

func TestGenerateOccurrences_Once(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := now.Add(7 * 24 * time.Hour)

	svc := &ScheduleMaterializationService{cfg: MaterializationConfig{}}

	t.Run("ONCE schedule within window produces one occurrence", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeOnce, nil, true, nil, nil)
		got, err := svc.generateOccurrences(s, now, windowEnd)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, s.EffectiveFrom, got[0].OccurrenceStart)
		assert.Equal(t, s.EffectiveUntil, got[0].OccurrenceEnd)
		assert.Equal(t, enums.OccurrenceStatusActive, got[0].Status)
		assert.Equal(t, enums.OccurrenceSourceRecurrence, got[0].Source)
	})

	t.Run("ONCE schedule entirely before window produces no occurrences", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeOnce, nil, true, nil, nil)
		// Move the schedule into the past.
		past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		s.EffectiveFrom = past
		s.EffectiveUntil = past.Add(time.Hour)

		got, err := svc.generateOccurrences(s, now, windowEnd)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("ONCE schedule starting after window produces no occurrences", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeOnce, nil, true, nil, nil)
		s.EffectiveFrom = windowEnd.Add(time.Hour)
		s.EffectiveUntil = windowEnd.Add(2 * time.Hour)

		got, err := svc.generateOccurrences(s, now, windowEnd)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestGenerateOccurrences_RRule(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := now.Add(7 * 24 * time.Hour)

	svc := &ScheduleMaterializationService{cfg: MaterializationConfig{}}

	t.Run("daily RRULE within 7-day window produces 7 occurrences", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeRRule, ptr("FREQ=DAILY;COUNT=100"), true, nil, nil)
		got, err := svc.generateOccurrences(s, now, windowEnd)
		require.NoError(t, err)
		assert.LessOrEqual(t, 7, len(got), "should have at least 7 daily occurrences")
	})

	t.Run("RRULE with TOD produces correct slot duration", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeRRule, ptr("FREQ=DAILY;COUNT=5"), false, ptr("09:00"), ptr("17:00"))
		got, err := svc.generateOccurrences(s, now, windowEnd)
		require.NoError(t, err)
		require.NotEmpty(t, got)
		for _, o := range got {
			dur := o.OccurrenceEnd.Sub(o.OccurrenceStart)
			assert.Equal(t, 8*time.Hour, dur, "slot should be 8 hours (09:00–17:00)")
		}
	})

	t.Run("RRULE nil rule returns error", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeRRule, nil, true, nil, nil)
		_, err := svc.generateOccurrences(s, now, windowEnd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no recurrence rule")
	})

	t.Run("invalid RRULE string returns error", func(t *testing.T) {
		t.Parallel()
		s := makeSchedule(enums.RecurrenceTypeRRule, ptr("NOT_VALID_RRULE"), true, nil, nil)
		_, err := svc.generateOccurrences(s, now, windowEnd)
		require.Error(t, err)
	})
}

func TestGenerateOccurrences_Calendar(t *testing.T) {
	t.Parallel()
	svc := &ScheduleMaterializationService{cfg: MaterializationConfig{}}

	now := time.Now().UTC()
	s := makeSchedule(enums.RecurrenceTypeCalendar, nil, true, nil, nil)
	got, err := svc.generateOccurrences(s, now, now.Add(time.Hour))
	require.NoError(t, err) // should not error – just skip gracefully
	assert.Empty(t, got)
}

func TestGenerateOccurrences_UnknownType(t *testing.T) {
	t.Parallel()
	svc := &ScheduleMaterializationService{cfg: MaterializationConfig{}}

	now := time.Now().UTC()
	s := makeSchedule("UNKNOWN", nil, true, nil, nil)
	_, err := svc.generateOccurrences(s, now, now.Add(time.Hour))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported recurrence type")
}

// ---------------------------------------------------------------------------
// slotDurationForSchedule unit tests
// ---------------------------------------------------------------------------

func TestSlotDurationForSchedule(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	tests := []struct {
		name     string
		allDay   bool
		todStart *string
		todEnd   *string
		want     time.Duration
		wantErr  bool
	}{
		{"all-day returns 24h", true, nil, nil, 24 * time.Hour, false},
		{"nil TOD returns 24h", false, nil, nil, 24 * time.Hour, false},
		{"09:00-17:00 = 8h", false, ptr("09:00"), ptr("17:00"), 8 * time.Hour, false},
		{"22:00-06:00 cross-midnight = 8h", false, ptr("22:00"), ptr("06:00"), 8 * time.Hour, false},
		{"bad start returns error", false, ptr("99:99"), ptr("17:00"), 0, true},
		{"bad end returns error", false, ptr("09:00"), ptr("99:99"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &entity.Schedule{AllDay: tt.allDay, TimeOfDayStart: tt.todStart, TimeOfDayEnd: tt.todEnd}
			got, err := slotDurationForSchedule(s, loc)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MaterializeWindow integration-style unit tests
// ---------------------------------------------------------------------------

func TestMaterializeWindow_HappyPath(t *testing.T) {
	t.Parallel()

	schedID := uuid.New()
	now := time.Now().UTC()
	sched := &entity.Schedule{
		RecurrenceType: enums.RecurrenceTypeOnce,
		EffectiveFrom:  now.Add(-time.Hour),
		EffectiveUntil: now.Add(time.Hour),
		IsActive:       true,
		Timezone:       "UTC",
	}
	sched.ID = schedID

	var upsertCalled bool
	var upsertedOccurrences []*entity.ScheduleOccurrence

	// Override ListActiveSchedulesInWindow
	occRepo := &mockOccurrenceRepository{
		upsertFn: func(_ context.Context, occ []*entity.ScheduleOccurrence) error {
			upsertCalled = true
			upsertedOccurrences = occ
			return nil
		},
	}

	svc := &ScheduleMaterializationService{
		scheduleRepo:   &listActiveWindowMock{schedules: []*entity.Schedule{sched}},
		occurrenceRepo: occRepo,
		cfg:            MaterializationConfig{WindowDuration: 7 * 24 * time.Hour},
	}

	err := svc.MaterializeWindow(context.Background())
	require.NoError(t, err)
	assert.True(t, upsertCalled, "upsert should have been called")
	require.Len(t, upsertedOccurrences, 1)
	assert.Equal(t, schedID, upsertedOccurrences[0].ScheduleID)
}

func TestMaterializeWindow_ScheduleRepoError(t *testing.T) {
	t.Parallel()

	svc := &ScheduleMaterializationService{
		scheduleRepo:   &listActiveWindowMock{err: errors.New("db timeout")},
		occurrenceRepo: &mockOccurrenceRepository{},
		cfg:            MaterializationConfig{},
	}

	err := svc.MaterializeWindow(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing active schedules")
}

func TestCleanupPastOccurrences(t *testing.T) {
	t.Parallel()

	var gotBefore time.Time
	occRepo := &mockOccurrenceRepository{
		deletePastFn: func(_ context.Context, before time.Time) error {
			gotBefore = before
			return nil
		},
	}

	svc := &ScheduleMaterializationService{
		occurrenceRepo: occRepo,
		cfg:            MaterializationConfig{RetentionPeriod: 30 * 24 * time.Hour},
	}

	err := svc.CleanupPastOccurrences(context.Background())
	require.NoError(t, err)

	// cutoff should be ~30 days before now
	expectedCutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	assert.WithinDuration(t, expectedCutoff, gotBefore, 5*time.Second)
}

// ---------------------------------------------------------------------------
// listActiveWindowMock — partial mock for ListActiveSchedulesInWindow
// ---------------------------------------------------------------------------

// listActiveWindowMock wraps mockScheduleRepository and overrides
// ListActiveSchedulesInWindow for the materialization tests.
type listActiveWindowMock struct {
	mockScheduleRepository
	schedules []*entity.Schedule
	err       error
}

func (m *listActiveWindowMock) ListActiveSchedulesInWindow(_ context.Context, _ time.Time) ([]*entity.Schedule, error) {
	return m.schedules, m.err
}

// CheckScheduleOverlap is required by the interface; not exercised here.
func (m *listActiveWindowMock) CheckScheduleOverlap(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) (*entity.Schedule, error) {
	return nil, nil
}
