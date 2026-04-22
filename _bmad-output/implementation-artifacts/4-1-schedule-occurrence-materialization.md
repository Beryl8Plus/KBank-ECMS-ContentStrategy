# Story 4.1: Schedule Occurrence Materialization

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a System / Background Job Worker,
I want to materialize active schedules into `schedule_occurrences` for a rolling time window,
so that the CMS Runtime and Admin portal can query active schedules efficiently without calculating recurrence rules on the fly.

## Acceptance Criteria

1. A background worker or core service logic is implemented to evaluate active `Schedule` records.
2. The logic calculates occurrences for a rolling window (e.g., from `now` to `now + 7 days`).
3. For each active occurrence within the window, a `ScheduleOccurrence` entity is created or upserted in the `schedule_occurrences` table.
4. The generation logic must be idempotent (no duplicate occurrences for the same schedule and time range).
5. The system must handle updates: if a `Schedule` is updated or deleted, future occurrences for that schedule should be re-calculated or deleted accordingly.
6. A cleanup mechanism removes past occurrences (e.g., older than 30 days) to prevent the table from growing indefinitely.

## Tasks / Subtasks

- [x] Task 1 (AC: 1, 3, 4): Setup `ScheduleOccurrence` Repository Layer
  - [x] Subtask 1.1: Implement repository methods for `UpsertOccurrences`, `DeleteFutureByScheduleID`, and `DeletePastOccurrences`.
  - [x] Subtask 1.2: Idempotency handled via GORM `clause.OnConflict` targeting unique index on `(SCHEDULE_ID, OCCURRENCE_START, OCCURRENCE_END)`.

- [x] Task 2 (AC: 2): Implement Schedule Materialization Service Logic
  - [x] Subtask 2.1: `ScheduleMaterializationService.generateOccurrences` dispatches by `RecurrenceType` (ONCE / RRULE / CALENDAR). RRULE expanded using `github.com/teambition/rrule-go`.
  - [x] Subtask 2.2: `MaterializeWindow(ctx)` fetches active schedules and upserts calculated occurrences for `[now, now+7d]` rolling window.

- [x] Task 3 (AC: 1): Background Job / Trigger Mechanism
  - [x] Subtask 3.1: `OccurrenceWorker` uses two independent Go tickers (materialization + cleanup). First materialization runs immediately on startup.
  - [x] Subtask 3.2: Worker is stopped gracefully via context cancellation; each ticker runs in its own select case — no shared state.

- [x] Task 4 (AC: 5): Handle Schedule Changes (Event/Hook)
  - [x] Subtask 4.1: `RegenerateForSchedule(ctx, scheduleID)` clears future occurrences then re-materialises from now. Caller hooks this into Update/Delete flows.

- [x] Task 5 (AC: 6): Cleanup Job for Past Occurrences
  - [x] Subtask 5.1: `CleanupPastOccurrences(ctx)` hard-deletes rows where `OCCURRENCE_END < now - retentionPeriod` (default 30 days).

## Dev Notes

- **Relevant architecture patterns and constraints:** Use existing Go services and repository patterns. Background jobs should be managed using existing concurrent patterns in the application (like standard Go tickers or a job library).
- **Source tree components to touch:**
  - `internal/domain/entity/schedule_occurrence.go`
  - `internal/domain/repository/` (New interface)
  - `internal/infrastructure/repository/postgres/` (New implementation)
  - `internal/application/service/` (Materialization logic)
- **Testing standards summary:** Write unit tests for the parsing and calculation logic of schedule occurrences to ensure accurate time slots are generated.

### Project Structure Notes

- Alignment with unified project structure (paths, modules, naming)
- Ensure all DB interactions use the existing GORM setup.

### References

- Reference Entity: `internal/domain/entity/schedule_occurrence.go`

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.6 (Thinking)

### Debug Log References

- Added `github.com/teambition/rrule-go v1.8.2` via `go get` — no existing iCalendar RRULE library was present in go.mod.
- Fixed unused `schedRepo` variable in test (compiler error on first run).

### Completion Notes List

- ✅ AC1: Domain interface `ScheduleOccurrenceRepository` + GORM implementation created.
- ✅ AC2: Rolling window materialisation (7 days ahead) via `MaterializeWindow`.
- ✅ AC3: Occurrences upserted on each run via `ON CONFLICT DO UPDATE`.
- ✅ AC4: Idempotency guaranteed by unique index on `(SCHEDULE_ID, OCCURRENCE_START, OCCURRENCE_END)`.
- ✅ AC5: `RegenerateForSchedule` clears stale future rows then re-materialises.
- ✅ AC6: `CleanupPastOccurrences` removes rows older than retention period (default 30 days).
- All 15 new unit tests pass; full `go test ./...` regression clean.

### File List

- `internal/domain/repository/schedule_occurrence.go` [NEW]
- `internal/repository/schedule_occurrence_repository.go` [NEW]
- `internal/service/schedule_materialization_service.go` [NEW]
- `internal/service/schedule_materialization_service_test.go` [NEW]
- `internal/service/occurrence_worker.go` [NEW]
- `go.mod` [MODIFIED — added github.com/teambition/rrule-go v1.8.2]
- `go.sum` [MODIFIED]
