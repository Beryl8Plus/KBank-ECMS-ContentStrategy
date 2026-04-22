package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/teambition/rrule-go"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"
)

// MaterializationConfig holds tunable parameters for the occurrence generator.
type MaterializationConfig struct {
	// WindowDuration is how far ahead to materialise occurrences from now.
	// Defaults to 7 days when zero.
	WindowDuration time.Duration

	// RetentionPeriod is how old an occurrence must be (past its end) before
	// the cleanup job removes it. Defaults to 30 days when zero.
	RetentionPeriod time.Duration
}

func (c *MaterializationConfig) windowDuration() time.Duration {
	if c.WindowDuration <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.WindowDuration
}

func (c *MaterializationConfig) retentionPeriod() time.Duration {
	if c.RetentionPeriod <= 0 {
		return 30 * 24 * time.Hour
	}
	return c.RetentionPeriod
}

// ScheduleMaterializationService materialises active schedules into
// ScheduleOccurrence rows so that the CMS Runtime can query them with a
// simple time-range predicate instead of evaluating recurrence rules on
// every request.
type ScheduleMaterializationService struct {
	scheduleRepo   domainrepo.ScheduleRepository
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository
	cfg            MaterializationConfig
}

// NewScheduleMaterializationService creates a new
// ScheduleMaterializationService.
func NewScheduleMaterializationService(
	scheduleRepo domainrepo.ScheduleRepository,
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository,
	cfg MaterializationConfig,
) *ScheduleMaterializationService {
	return &ScheduleMaterializationService{
		scheduleRepo:   scheduleRepo,
		occurrenceRepo: occurrenceRepo,
		cfg:            cfg,
	}
}

// MaterializeWindow fetches all active schedules and materialises occurrences
// for the rolling window [now, now + windowDuration).
// This method is idempotent: re-running it for the same window produces no
// duplicate rows thanks to the ON CONFLICT upsert in the repository.
func (s *ScheduleMaterializationService) MaterializeWindow(ctx context.Context) error {
	now := time.Now().UTC()
	windowEnd := now.Add(s.cfg.windowDuration())

	// Fetch every schedule whose effective period overlaps with the window.
	// ListActiveSchedulesInWindow returns schedules active at `now`; we rely
	// on EffectiveUntil to cap generation at the schedule's own end date.
	schedules, err := s.scheduleRepo.ListActiveSchedulesInWindow(ctx, now)
	if err != nil {
		return fmt.Errorf("materialization: listing active schedules: %w", err)
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "MATERIALIZATION",
		Level:   "INFO",
		Message: fmt.Sprintf("materializing schedule occurrences (schedules=%d window_start=%s window_end=%s)",
			len(schedules), now.Format(time.RFC3339), windowEnd.Format(time.RFC3339)),
	})

	for _, sched := range schedules {
		occurrences, err := s.generateOccurrences(sched, now, windowEnd)
		if err != nil {
			// Log and continue — one bad schedule must not abort the whole job.
			logger.LSystem(ctx, entity.SystemLog{
				Service: "MATERIALIZATION",
				Level:   "ERROR",
				Message: fmt.Sprintf("failed to generate occurrences for schedule %s: %s", sched.ID, err.Error()),
			})
			continue
		}

		if len(occurrences) == 0 {
			continue
		}

		if err := s.occurrenceRepo.UpsertOccurrences(ctx, occurrences); err != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "MATERIALIZATION",
				Level:   "ERROR",
				Message: fmt.Sprintf("failed to upsert occurrences for schedule %s: %s", sched.ID, err.Error()),
			})
		}
	}

	return nil
}

// RegenerateForSchedule clears all future occurrences for the given schedule
// and re-materialises them from `now`. This is the hook called when a
// Schedule is updated or (soft-)deleted so that the occurrence table stays
// consistent with the source Schedule.
func (s *ScheduleMaterializationService) RegenerateForSchedule(
	ctx context.Context,
	scheduleID uuid.UUID,
) error {
	now := time.Now().UTC()

	// 1. Clear stale future occurrences.
	if err := s.occurrenceRepo.DeleteFutureByScheduleID(ctx, scheduleID, now); err != nil {
		return fmt.Errorf("regenerating: clear future occurrences: %w", err)
	}

	// 2. Reload the schedule (may be nil if it was deleted).
	sched, err := s.scheduleRepo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("regenerating: get schedule: %w", err)
	}
	if sched == nil || !sched.IsActive {
		// Schedule was deleted or deactivated — clearing is sufficient.
		return nil
	}

	// 3. Re-materialise.
	windowEnd := now.Add(s.cfg.windowDuration())
	occurrences, err := s.generateOccurrences(sched, now, windowEnd)
	if err != nil {
		return fmt.Errorf("regenerating: generate occurrences: %w", err)
	}

	if len(occurrences) == 0 {
		return nil
	}

	if err := s.occurrenceRepo.UpsertOccurrences(ctx, occurrences); err != nil {
		return fmt.Errorf("regenerating: upsert occurrences: %w", err)
	}

	return nil
}

// CleanupPastOccurrences deletes all occurrences older than the configured
// retention period. Should be called periodically (e.g. daily).
func (s *ScheduleMaterializationService) CleanupPastOccurrences(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-s.cfg.retentionPeriod())
	if err := s.occurrenceRepo.DeletePastOccurrences(ctx, cutoff); err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}
	logger.LSystem(ctx, entity.SystemLog{
		Service: "MATERIALIZATION",
		Level:   "INFO",
		Message: fmt.Sprintf("past occurrences cleaned up (cutoff=%s)", cutoff.Format(time.RFC3339)),
	})
	return nil
}

// ---------------------------------------------------------------------------
// Occurrence generation helpers
// ---------------------------------------------------------------------------

// generateOccurrences expands a single Schedule into concrete time slots
// within [windowStart, windowEnd).
//
// Three recurrence strategies are supported:
//   - ONCE    – a single slot spanning [EffectiveFrom, EffectiveUntil).
//   - RRULE   – iCalendar RRULE string; each hit produces a slot of duration
//     equal to (TimeOfDayEnd – TimeOfDayStart) or a full day.
//   - CALENDAR – not yet supported; returns an empty slice.
func (s *ScheduleMaterializationService) generateOccurrences(
	sched *entity.Schedule,
	windowStart, windowEnd time.Time,
) ([]*entity.ScheduleOccurrence, error) {
	switch sched.RecurrenceType {
	case enums.RecurrenceTypeOnce:
		return s.generateOnceOccurrence(sched, windowStart, windowEnd), nil
	case enums.RecurrenceTypeRRule:
		return s.generateRRuleOccurrences(sched, windowStart, windowEnd)
	case enums.RecurrenceTypeCalendar:
		// CALENDAR-based recurrence requires joining CalendarDate rows.
		// This is not yet implemented; return empty to skip gracefully.
		logger.LSystem(context.Background(), entity.SystemLog{
			Service: "MATERIALIZATION",
			Level:   "WARN",
			Message: fmt.Sprintf("CALENDAR recurrence type not yet supported for materialisation (schedule_id=%s)", sched.ID),
		})
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported recurrence type %q for schedule %s", sched.RecurrenceType, sched.ID)
	}
}

// generateOnceOccurrence creates at most one occurrence for a ONCE schedule.
// The slot is [EffectiveFrom, EffectiveUntil) capped to the window.
func (s *ScheduleMaterializationService) generateOnceOccurrence(
	sched *entity.Schedule,
	windowStart, windowEnd time.Time,
) []*entity.ScheduleOccurrence {
	start := sched.EffectiveFrom
	end := sched.EffectiveUntil

	// Slot must overlap the window.
	if end.IsZero() || !start.Before(windowEnd) || !end.After(windowStart) {
		return nil
	}

	return []*entity.ScheduleOccurrence{
		s.newOccurrence(sched.ID, start, end, enums.OccurrenceSourceRecurrence),
	}
}

// generateRRuleOccurrences expands an RRULE string into occurrence slots
// within the given window.
//
// Each hit of the rule is turned into a slot whose width equals the
// TimeOfDay duration (TimeOfDayStart → TimeOfDayEnd) if both are set, or
// exactly 24 hours for all-day events. The slot start and end are computed
// in the schedule's Timezone.
func (s *ScheduleMaterializationService) generateRRuleOccurrences(
	sched *entity.Schedule,
	windowStart, windowEnd time.Time,
) ([]*entity.ScheduleOccurrence, error) {
	if sched.RecurrenceRule == nil || *sched.RecurrenceRule == "" {
		return nil, fmt.Errorf("RRULE schedule %s has no recurrence rule", sched.ID)
	}

	loc, err := time.LoadLocation(sched.Timezone)
	if err != nil {
		// Fallback to UTC so a bad TZ doesn't abort the whole job.
		logger.LSystem(context.Background(), entity.SystemLog{
			Service: "MATERIALIZATION",
			Level:   "WARN",
			Message: fmt.Sprintf("unknown timezone %q for schedule %s, falling back to UTC", sched.Timezone, sched.ID),
		})
		loc = time.UTC
	}

	// Parse the RRULE.  The library expects a full RRULE: prefix or a bare
	// property value — normalise to the bare-value form.
	ruleStr := *sched.RecurrenceRule
	ruleStr = strings.TrimPrefix(ruleStr, "RRULE:")

	rule, err := rrule.StrToRRule(ruleStr)
	if err != nil {
		return nil, fmt.Errorf("parsing RRULE for schedule %s: %w", sched.ID, err)
	}

	// Determine the duration of a single slot.
	slotDuration, err := slotDurationForSchedule(sched, loc)
	if err != nil {
		return nil, fmt.Errorf("computing slot duration for schedule %s: %w", sched.ID, err)
	}

	// Respect the schedule's own effective period.
	genStart := windowStart
	if sched.EffectiveFrom.After(genStart) {
		genStart = sched.EffectiveFrom
	}
	genEnd := windowEnd
	if !sched.EffectiveUntil.IsZero() && sched.EffectiveUntil.Before(genEnd) {
		genEnd = sched.EffectiveUntil
	}

	// Set the RRULE's DTSTART so the expansion is anchored correctly.
	dtStart := sched.EffectiveFrom.In(loc)
	rule.DTStart(dtStart)

	hits := rule.Between(genStart.In(loc), genEnd.In(loc), true /* inclusive */)

	occurrences := make([]*entity.ScheduleOccurrence, 0, len(hits))
	for _, hit := range hits {
		start := hit.UTC()
		end := start.Add(slotDuration)
		// Clamp to the schedule's effective until.
		if !sched.EffectiveUntil.IsZero() && end.After(sched.EffectiveUntil) {
			end = sched.EffectiveUntil
		}
		occurrences = append(occurrences, s.newOccurrence(sched.ID, start, end, enums.OccurrenceSourceRecurrence))
	}

	return occurrences, nil
}

// slotDurationForSchedule computes the duration of a single occurrence slot.
//
// If both TimeOfDayStart and TimeOfDayEnd are set (format "HH:MM"), the
// duration is their difference. For all-day schedules the duration is 24 h.
func slotDurationForSchedule(sched *entity.Schedule, loc *time.Location) (time.Duration, error) {
	if sched.AllDay || sched.TimeOfDayStart == nil || sched.TimeOfDayEnd == nil {
		return 24 * time.Hour, nil
	}

	parseTime := func(s string) (time.Time, error) {
		return time.ParseInLocation("15:04", s, loc)
	}

	startT, err := parseTime(*sched.TimeOfDayStart)
	if err != nil {
		return 0, fmt.Errorf("invalid TimeOfDayStart %q: %w", *sched.TimeOfDayStart, err)
	}
	endT, err := parseTime(*sched.TimeOfDayEnd)
	if err != nil {
		return 0, fmt.Errorf("invalid TimeOfDayEnd %q: %w", *sched.TimeOfDayEnd, err)
	}

	d := endT.Sub(startT)
	if d <= 0 {
		// Cross-midnight slot: add 24 h.
		d += 24 * time.Hour
	}
	return d, nil
}

// newOccurrence is a factory helper that builds a ScheduleOccurrence.
func (s *ScheduleMaterializationService) newOccurrence(
	scheduleID uuid.UUID,
	start, end time.Time,
	source enums.OccurrenceSource,
) *entity.ScheduleOccurrence {
	return &entity.ScheduleOccurrence{
		ScheduleID:      scheduleID,
		OccurrenceStart: start,
		OccurrenceEnd:   end,
		Status:          enums.OccurrenceStatusActive,
		Source:          source,
	}
}
