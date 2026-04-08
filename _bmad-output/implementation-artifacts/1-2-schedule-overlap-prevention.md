# Story 1.2: Schedule Overlap Prevention — DB Constraint + Service Validation

Status: review

## Story

As a **backend developer**,
I want to **add a PostgreSQL EXCLUDE constraint and a service-layer overlap check so that no two active schedules for the same placement can have overlapping effective date ranges**,
so that **the system enforces data integrity at both the database and application levels (defense-in-depth), preventing conflicting schedules from being persisted**.

## Acceptance Criteria

1. **AC-1: PostgreSQL EXCLUDE constraint prevents overlapping schedules at DB level**
   - Given the `schedules` table with `btree_gist` extension enabled
   - When an INSERT or UPDATE would create two active schedules for the same `placement_id` with overlapping `[effective_from, effective_until)` ranges
   - Then the database rejects the operation with a constraint violation
   - And soft-deleted records (`deleted_at IS NOT NULL`) are excluded from the constraint

2. **AC-2: Service-layer validation returns clear error before DB insert**
   - Given a new or updated Schedule with `placement_id`, `effective_from`, `effective_until`
   - When another active schedule exists for the same `placement_id` with an overlapping date range
   - Then the service returns a descriptive error (e.g., "schedule overlaps with existing schedule [ID] for placement [ID] from [date] to [date]")
   - And the error is returned before attempting the DB insert

3. **AC-3: Migration script enables btree_gist and adds EXCLUDE constraint**
   - Given the migration tool (`cmd/migrate/main.go`)
   - When migration runs
   - Then `CREATE EXTENSION IF NOT EXISTS btree_gist` executes first
   - And the EXCLUDE constraint `no_overlap_active_schedule_per_placement` is added to the `schedules` table
   - And the migration is idempotent (safe to run multiple times)

4. **AC-4: Overlap check correctly handles edge cases**
   - Given various boundary scenarios
   - When checking for overlaps:
     - Adjacent ranges (A ends exactly when B starts) → allowed (no overlap)
     - Same placement, one schedule inactive (`is_active = false`) → allowed
     - Same placement, schedule soft-deleted → allowed
     - Different placements, same date range → allowed
     - Updating own record (same ID) → excluded from overlap check
   - Then each scenario is handled correctly

5. **AC-5: All existing and new tests pass**
   - Given the new overlap validation logic and DB constraint
   - When the test suite runs
   - Then unit tests for the service-layer overlap check pass
   - And no existing tests regress

## Tasks / Subtasks

- [x] **Task 1: Add btree_gist extension and EXCLUDE constraint migration** (AC: #1, #3)
  - [x] 1.1 Add post-AutoMigrate raw SQL in `cmd/migrate/main.go` to enable `btree_gist` extension
  - [x] 1.2 Add raw SQL to create EXCLUDE constraint `no_overlap_active_schedule_per_placement` using `placement_id WITH =` and `tstzrange(effective_from, effective_until) WITH &&` where `is_active = true AND deleted_at IS NULL`
  - [x] 1.3 Make migration idempotent — use `IF NOT EXISTS` / check `pg_constraint` before adding

- [x] **Task 2: Create schedule repository for overlap check** (AC: #2, #4)
  - [x] 2.1 Create `internal/domain/repository/schedule.go` — define `ScheduleRepository` interface with `CheckScheduleOverlap(ctx context.Context, placementID uuid.UUID, effectiveFrom, effectiveUntil time.Time, excludeID *uuid.UUID) (*entity.Schedule, error)`
  - [x] 2.2 Create `internal/repository/schedule_repository.go` — implement `SchedulePostgresRepository` with `NewSchedulePostgresRepository(db *gorm.DB)` constructor, compile-time check `var _ domainrepo.ScheduleRepository = (*SchedulePostgresRepository)(nil)`, and `CheckScheduleOverlap` querying active non-deleted schedules with overlapping date range, excluding `excludeID` when provided

- [x] **Task 3: Create schedule service with overlap validation** (AC: #2, #4)
  - [x] 3.1 Create `internal/service/schedule_service.go` with `ScheduleService` struct holding a `ScheduleRepository` dependency (from `internal/domain/repository/schedule.go`)
  - [x] 3.2 Implement `ValidateScheduleOverlap(ctx context.Context, schedule *entity.Schedule) error` that calls repository overlap check and returns descriptive error with conflicting schedule details

- [x] **Task 4: Write unit tests for overlap validation** (AC: #4, #5)
  - [x] 4.1 Create `internal/service/schedule_service_test.go` — inject a local mock struct that implements `ScheduleRepository` (one method, no external libraries needed). Table-driven tests covering: overlap detected, no overlap, adjacent ranges allowed, inactive schedule allowed, self-update excluded, different placement allowed
  - [x] 4.2 Create `internal/repository/schedule_repository_test.go` — use GORM DryRun mode (`&gorm.Config{DryRun: true}`) to verify query structure. Test that excludeID filter is conditionally applied.

- [x] **Task 5: Verify compilation and run full test suite** (AC: #5)
  - [x] 5.1 Run `go build ./...` to verify no compilation errors
  - [x] 5.2 Run full test suite to ensure no regressions

## Dev Notes

### Source Reference

- **Story 1.1 (prerequisite):** `_bmad-output/implementation-artifacts/1-1-schedule-entity-refactor.md`
- **Schedule entity:** `internal/domain/entity/schedule.go`
- **Migration entry point:** `cmd/migrate/main.go`
- **Schedule repository interface (CREATE):** `internal/domain/repository/schedule.go`
- **Schedule repository implementation (CREATE):** `internal/repository/schedule_repository.go`
- **DO NOT MODIFY:** `internal/domain/repository/database.go`, `internal/repository/postgres_repository.go`

### Project Conventions (MUST FOLLOW)

**Per-domain repository pattern** — each domain has its OWN interface file + concrete implementation. NEVER add schedule methods to `DatabaseRepository`.

| Domain | Interface file | Implementation file |
|--------|---------------|---------------------|
| database/migrate | `internal/domain/repository/database.go` | `internal/repository/postgres_repository.go` |
| permission | `internal/domain/repository/permission.go` | `internal/repository/permission_repository.go` |
| cache | `internal/domain/repository/cache.go` | `internal/repository/redis_repository.go` |
| storage | `internal/domain/repository/storage.go` | `internal/repository/azure_repository.go` |
| **schedule (CREATE)** | `internal/domain/repository/schedule.go` | `internal/repository/schedule_repository.go` |

Every implementation file MUST include a compile-time interface check:
```go
var _ domainrepo.ScheduleRepository = (*SchedulePostgresRepository)(nil)
```

**Service pattern** — services in `internal/service/` inject the domain-specific repository interface:
```go
type ScheduleService struct { repo domainrepo.ScheduleRepository }
func NewScheduleService(repo domainrepo.ScheduleRepository) *ScheduleService {
    return &ScheduleService{repo: repo}
}
```

**Error wrapping** — always `fmt.Errorf("context message: %w", err)` for `errors.Is`/`errors.As` compatibility.

**Testing pattern** — table-driven tests with `t.Run`, `testify/assert` for assertions, `require.NoError` for setup. Mock `ScheduleRepository` with a local struct for service tests.

**GORM soft-delete** — `BaseModel` embeds `gorm.DeletedAt`. Standard GORM queries auto-add `deleted_at IS NULL`. The EXCLUDE constraint MUST explicitly include `deleted_at IS NULL` in its `WHERE` clause.

### PostgreSQL EXCLUDE Constraint Design

```sql
-- Step 1: Enable extension (idempotent)
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Step 2: Add EXCLUDE constraint (check existence first)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'no_overlap_active_schedule_per_placement'
  ) THEN
    ALTER TABLE schedules
    ADD CONSTRAINT no_overlap_active_schedule_per_placement
    EXCLUDE USING gist (
      placement_id WITH =,
      tstzrange(effective_from, effective_until) WITH &&
    ) WHERE (is_active = true AND deleted_at IS NULL);
  END IF;
END $$;
```

**Key design decisions:**
- Uses half-open range `[effective_from, effective_until)` via `tstzrange()` default — adjacent ranges don't overlap
- Only applies to `is_active = true AND deleted_at IS NULL` — inactive/deleted schedules are exempt
- Constraint name `no_overlap_active_schedule_per_placement` is descriptive for error messages

### Service-Layer Overlap Query

```go
// Overlap detection: two ranges [A_from, A_until) and [B_from, B_until) overlap when:
//   A_from < B_until AND A_until > B_from
func (r *SchedulePostgresRepository) CheckScheduleOverlap(
    ctx context.Context,
    placementID uuid.UUID,
    effectiveFrom, effectiveUntil time.Time,
    excludeID *uuid.UUID,
) (*entity.Schedule, error) {
    var conflict entity.Schedule
    q := r.db.WithContext(ctx).
        Where("placement_id = ?", placementID).
        Where("is_active = true").
        Where("effective_from < ? AND effective_until > ?", effectiveUntil, effectiveFrom)
    if excludeID != nil {
        q = q.Where("id != ?", *excludeID)
    }
    err := q.First(&conflict).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil // no overlap
    }
    return &conflict, err
}
```

**Note:** GORM automatically adds `deleted_at IS NULL` to queries because `BaseModel` includes `gorm.DeletedAt`. No need to add it explicitly.

### Migration Integration Point

The raw SQL must run **after** `db.AutoMigrate(models...)` in `cmd/migrate/main.go` since the `schedules` table must exist first. Pattern:

```go
// After AutoMigrate succeeds...
sqlDB, err := db.DB()
if err != nil {
    log.Fatal(fmt.Errorf("failed to get sql.DB: %w", err))
}
if _, err = sqlDB.Exec(`CREATE EXTENSION IF NOT EXISTS btree_gist`); err != nil {
    log.Fatal(fmt.Errorf("failed to enable btree_gist extension: %w", err))
}
if _, err = sqlDB.Exec(`DO $$ ... END $$;`); err != nil {
    log.Fatal(fmt.Errorf("failed to add EXCLUDE constraint: %w", err))
}
```

### Edge Case: `tstzrange` behavior

| Scenario | Result |
|---|---|
| Schedule A: May 1 – May 15, Schedule B: May 15 – May 31 | ✅ Allowed (half-open: May 15 is start of B, not in A) |
| Schedule A: May 1 – May 20, Schedule B: May 15 – May 31 | ❌ Blocked (overlaps May 15–20) |
| Schedule A active, Schedule B inactive, same range | ✅ Allowed (constraint only active) |
| Schedule A deleted (soft), Schedule B same range | ✅ Allowed (constraint filters deleted_at IS NULL) |

### Handling EXCLUDE Constraint Violation (C3: Defense-in-Depth)

When two goroutines pass the service check simultaneously (race condition), the DB constraint fires. GORM wraps the Postgres error — detect it explicitly:

```go
import (
    "errors"
    "github.com/jackc/pgx/v5/pgconn"
)

// isExclusionViolation returns true when err is a PostgreSQL exclusion_violation (23P01).
func isExclusionViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}
```

In the service's `CreateSchedule` / `UpdateSchedule` (future story), wrap DB errors:
```go
if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
    if isExclusionViolation(err) {
        return fmt.Errorf("schedule overlaps with an existing active schedule for placement %s: %w", schedule.PlacementID, err)
    }
    return fmt.Errorf("failed to save schedule: %w", err)
}
```

**Note:** `pgx/v5` is the driver used by `gorm.io/driver/postgres` v1.6.0. The `pgconn` package is already in the module graph via the GORM Postgres driver — no new dependency needed.

### Critical Design Decisions

1. **Defense-in-depth** — DB constraint catches race conditions; service validation gives user-friendly errors
2. **Half-open intervals** — `tstzrange(from, until)` defaults to `[from, until)` which correctly allows adjacent ranges
3. **is_active filter** — inactive schedules are draft/paused and should not block new active ones
4. **excludeID for updates** — when updating a schedule, exclude its own ID from the overlap check
5. **Race condition error surfacing** — detect `pgconn.PgError` code `23P01` to return a clean error instead of leaking raw DB errors
6. **This story is overlap-prevention only** — no CRUD endpoints, no RRULE parsing, no frontend

### Files to Create/Modify

| Action | File Path |
|--------|----------|
| MODIFY | `cmd/migrate/main.go` |
| CREATE | `internal/domain/repository/schedule.go` |
| CREATE | `internal/repository/schedule_repository.go` |
| CREATE | `internal/repository/schedule_repository_test.go` |
| CREATE | `internal/service/schedule_service.go` |
| CREATE | `internal/service/schedule_service_test.go` |

### Out of Scope (future stories)

- Schedule CRUD API endpoints
- RRULE parsing and occurrence generation
- FullCalendar event API
- TimeOfDay overlap detection (only date-range overlap in this story)
- Calendar-based schedule overlap (calendar dates are resolved at occurrence level)

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.6 (GitHub Copilot)

### Debug Log References

N/A — no blocking issues encountered.

### Completion Notes List

- Story 1-2 overlap-prevention tasks (Tasks 1–5) were already implemented on disk upon inspection; checkboxes were updated to reflect actual state.
- Extended the implementation beyond story scope (as explicitly requested by user) to add full Schedule CRUD API: repository CRUD methods, service CRUD methods, DTOs, HTTP handler, and router wiring.
- Introduced `scheduleServicer` interface in the handler package to enable proper unit testing without pulling in concrete service/repository dependencies.
- 45 tests pass across handler, service, and repository packages.

### File List

| Action | File Path |
|--------|----------|
| MODIFIED | `cmd/migrate/main.go` |
| CREATED | `internal/domain/repository/schedule.go` |
| CREATED | `internal/repository/schedule_repository.go` |
| CREATED | `internal/repository/schedule_repository_test.go` |
| CREATED | `internal/service/schedule_service.go` |
| CREATED | `internal/service/schedule_service_test.go` |
| CREATED | `internal/delivery/http/dto/schedule_dto.go` |
| CREATED | `internal/delivery/http/handler/schedule_handler.go` |
| CREATED | `internal/delivery/http/handler/schedule_handler_test.go` |
| MODIFIED | `internal/delivery/http/router.go` |

### Change Log

- Extended `ScheduleRepository` interface with 5 CRUD methods (CreateSchedule, GetScheduleByID, ListSchedules, UpdateSchedule, DeleteSchedule)
- Implemented CRUD methods on `SchedulePostgresRepository`
- Extended `ScheduleService` with 5 CRUD methods (overlap validation on create/update)
- Created `schedule_dto.go`: `CreateScheduleRequest`, `UpdateScheduleRequest`, `ScheduleResponse`, `ToScheduleResponse`
- Created `schedule_handler.go`: 5 Gin endpoint handlers with Swagger annotations and custom response headers
- Introduced `scheduleServicer` interface in `schedule_handler.go` for testability
- Created `schedule_handler_test.go`: 16 tests covering all endpoints and error paths
- Wired `SchedulePostgresRepository → ScheduleService → ScheduleHandler` in `router.go` with route group `/schedules`
