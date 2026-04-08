# Scheduling & Calendar Models — Deep Dive Documentation

**Generated:** 2026-04-08
**Scope:** `internal/domain/entity/` (scheduling & calendar area)
**Files Analyzed:** 9
**Lines of Code:** ~265
**Workflow Mode:** Exhaustive Deep-Dive

---

## Overview

**Purpose:**  
Refactors the `schedules` table from a simple start/end timestamp pair into a three-table cron-based architecture capable of representing all four scheduling use cases required by the EMS business domain.

**Key Responsibilities:**

- Represent recurring schedules via RFC 5545 RRULE strings (e.g., weekday banners)
- Support calendar-linked date sources (public holidays, personal birthdays)
- Materialize schedule instances into `schedule_occurrences` for audit and FullCalendar display
- Provide strongly typed enum values validated at the Go application layer

**Integration Points:**

- `internal/domain/entity/models.go` — GORM `AutoMigrate` order (dependency-safe)
- `internal/domain/entity/enums/` — 4 new enum types consumed by Schedule, ScheduleOccurrence, Calendar
- Future: `ScheduleService` (occurrence generation), `FullCalendar` API endpoint (not in scope for this story)

---

## Complete File Inventory

### `internal/domain/entity/enums/recurrence_type.go`

**Purpose:** Defines the scheduling recurrence strategy type. Controls how the `schedules` table generates or links its occurrences.

**Lines of Code:** ~36

**What Future Contributors Must Know:**  
Adding a new recurrence type requires updating `IsValid()`, `ParseRecurrenceType()`, and any service-layer switch statements that branch on `RecurrenceType`. Do not accept raw strings from API input without calling `ParseRecurrenceType()`.

**Exports:**

| Symbol                                                  | Kind   | Description                                                                                   |
| ------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------- |
| `RecurrenceType`                                        | type   | `string` alias for safe enum usage                                                            |
| `RecurrenceTypeOnce`                                    | const  | `"ONCE"` — schedule fires exactly once in the given `effective_from`/`effective_until` window |
| `RecurrenceTypeRRule`                                   | const  | `"RRULE"` — schedule repeats per RFC 5545 RRULE string stored in `recurrence_rule`            |
| `RecurrenceTypeCalendar`                                | const  | `"CALENDAR"` — schedule follows dates from a linked `Calendar` entity                         |
| `(r RecurrenceType) String() string`                    | method | Returns raw string value                                                                      |
| `(r RecurrenceType) IsValid() bool`                     | method | Returns `true` if value is a known constant                                                   |
| `ParseRecurrenceType(s string) (RecurrenceType, error)` | func   | Parses and validates a raw string; use at API boundary                                        |

**Dependencies:** `fmt` (standard library only)

**Used By:**

- `internal/domain/entity/schedule.go` — `RecurrenceType` field type

**Key Implementation Details:**

```go
type RecurrenceType string

const (
    RecurrenceTypeOnce     RecurrenceType = "ONCE"
    RecurrenceTypeRRule    RecurrenceType = "RRULE"
    RecurrenceTypeCalendar RecurrenceType = "CALENDAR"
)
```

**Side Effects:** None — pure value type with no I/O  
**Error Handling:** `ParseRecurrenceType` returns `fmt.Errorf` with explicit valid values listed  
**Testing:** `internal/domain/entity/enums/recurrence_type_test.go` — covers `IsValid`, `ParseRecurrenceType`, `String` (all PASS)

---

### `internal/domain/entity/enums/occurrence_status.go`

**Purpose:** Defines the lifecycle status of a materialized `ScheduleOccurrence` instance.

**Lines of Code:** ~32

**What Future Contributors Must Know:**  
`ACTIVE` is the initial state. A cancelled occurrence should never be re-activated — create a new occurrence instead. `MODIFIED` indicates manual override of auto-generated time window.

**Exports:**

| Symbol                                                      | Kind   | Description                                                              |
| ----------------------------------------------------------- | ------ | ------------------------------------------------------------------------ |
| `OccurrenceStatus`                                          | type   | `string` alias                                                           |
| `OccurrenceStatusActive`                                    | const  | `"ACTIVE"` — occurrence is live and valid                                |
| `OccurrenceStatusCancelled`                                 | const  | `"CANCELLED"` — occurrence was voided                                    |
| `OccurrenceStatusModified`                                  | const  | `"MODIFIED"` — start/end was manually adjusted from auto-generated value |
| `(o OccurrenceStatus) String() string`                      | method | Returns raw string                                                       |
| `(o OccurrenceStatus) IsValid() bool`                       | method | Validates against known constants                                        |
| `ParseOccurrenceStatus(s string) (OccurrenceStatus, error)` | func   | Parses raw string with validation                                        |

**Dependencies:** `fmt`

**Used By:**

- `internal/domain/entity/schedule_occurrence.go` — `Status` field type

**Side Effects:** None  
**Testing:** `internal/domain/entity/enums/occurrence_status_test.go` — all PASS

---

### `internal/domain/entity/enums/occurrence_source.go`

**Purpose:** Records the origin of how a `ScheduleOccurrence` was generated.

**Lines of Code:** ~32

**What Future Contributors Must Know:**  
`RECURRENCE` = auto-generated by cron from RRULE. `CALENDAR` = auto-generated by cron from calendar dates. `MANUAL` = directly inserted by a human or admin API call. The `source` field is write-once — do not change after creation.

**Exports:**

| Symbol                                                      | Kind  | Description                                        |
| ----------------------------------------------------------- | ----- | -------------------------------------------------- |
| `OccurrenceSource`                                          | type  | `string` alias                                     |
| `OccurrenceSourceRecurrence`                                | const | `"RECURRENCE"` — generated from RRULE pattern      |
| `OccurrenceSourceCalendar`                                  | const | `"CALENDAR"` — generated from calendar date lookup |
| `OccurrenceSourceManual`                                    | const | `"MANUAL"` — inserted directly by operator         |
| `ParseOccurrenceSource(s string) (OccurrenceSource, error)` | func  | Boundary validator                                 |

**Dependencies:** `fmt`

**Used By:**

- `internal/domain/entity/schedule_occurrence.go` — `Source` field type

**Side Effects:** None  
**Testing:** `internal/domain/entity/enums/occurrence_source_test.go` — all PASS

---

### `internal/domain/entity/enums/calendar_type.go`

**Purpose:** Classifies the purpose of a `Calendar` master record.

**Lines of Code:** ~32

**What Future Contributors Must Know:**  
`HOLIDAY` calendars are typically managed by admin (e.g., Thai bank holiday list). `PERSONAL` calendars are user-managed (birthdays, etc.). `CUSTOM` is for project-specific date lists. Enum controls what UI labels and import sources are available.

**Exports:**

| Symbol                                              | Kind  | Description                                |
| --------------------------------------------------- | ----- | ------------------------------------------ |
| `CalendarType`                                      | type  | `string` alias                             |
| `CalendarTypeHoliday`                               | const | `"HOLIDAY"` — official holiday calendar    |
| `CalendarTypePersonal`                              | const | `"PERSONAL"` — user-defined personal dates |
| `CalendarTypeCustom`                                | const | `"CUSTOM"` — project-defined custom dates  |
| `ParseCalendarType(s string) (CalendarType, error)` | func  | Boundary validator                         |

**Dependencies:** `fmt`

**Used By:**

- `internal/domain/entity/calendar.go` — `Type` field type

**Side Effects:** None  
**Testing:** `internal/domain/entity/enums/calendar_type_test.go` — all PASS

---

### `internal/domain/entity/calendar.go`

**Purpose:** Master record for a named calendar source. Schedules with `recurrence_type = CALENDAR` point to this entity.

**Lines of Code:** ~12

**What Future Contributors Must Know:**  
`Calendar` has no date data itself — dates live in `CalendarDate`. Delete or deactivate a Calendar with care: all linked `Schedule` records and `CalendarDate` entries still reference it. Soft-delete (from `BaseModel`) is supported.

**Exports:**

| Symbol     | Kind   | Description                    |
| ---------- | ------ | ------------------------------ |
| `Calendar` | struct | GORM model; table: `calendars` |

**Fields:**

| Field       | DB Column   | Type           | GORM Tag                           | JSON       |
| ----------- | ----------- | -------------- | ---------------------------------- | ---------- |
| `BaseModel` | (embedded)  | —              | UUID PK, audit fields, soft-delete | —          |
| `Name`      | `name`      | `varchar(255)` | `size:255`                         | `name`     |
| `Type`      | `type`      | `varchar(255)` | `size:255`                         | `type`     |
| `IsActive`  | `is_active` | `boolean`      | `default:true`                     | `isActive` |

**Dependencies:**

- `kbank-ecms/internal/domain/entity/enums` — `CalendarType`

**Used By:**

- `internal/domain/entity/calendar_date.go` — FK `CalendarID`
- `internal/domain/entity/schedule.go` — nullable FK `CalendarID`
- `internal/domain/entity/models.go` — registered in `AllModels()`

**Side Effects:** None beyond GORM auto-migration (creates `calendars` table)

---

### `internal/domain/entity/calendar_date.go`

**Purpose:** Stores individual date entries belonging to a Calendar. Powers date lookup for `CALENDAR`-type schedules.

**Lines of Code:** ~20

**What Future Contributors Must Know:**  
`Date` uses `gorm:"type:date"` which maps to PostgreSQL native `date` (no time component). Go uses `time.Time` but GORM will zero out the time portion on save. `IsRecurring = true` means the date repeats every year (e.g., Songkran on April 13). The cron job (future story) must respect this when expanding dates.

**Exports:**

| Symbol         | Kind   | Description                         |
| -------------- | ------ | ----------------------------------- |
| `CalendarDate` | struct | GORM model; table: `calendar_dates` |

**Fields:**

| Field         | DB Column      | Type            | GORM Tag                | JSON                 |
| ------------- | -------------- | --------------- | ----------------------- | -------------------- |
| `BaseModel`   | (embedded)     | —               | UUID PK, audit fields   | —                    |
| `CalendarID`  | `calendar_id`  | `uuid NOT NULL` | `type:uuid;not null`    | `calendarId`         |
| `Calendar`    | —              | `*Calendar`     | `foreignKey:CalendarID` | `calendar,omitempty` |
| `Date`        | `date`         | `date`          | `type:date`             | `date`               |
| `Name`        | `name`         | `varchar(255)`  | `size:255`              | `name`               |
| `IsRecurring` | `is_recurring` | `boolean`       | `default:false`         | `isRecurring`        |

**Dependencies:**

- `time` — for `time.Time`
- `github.com/google/uuid` — for FK ID type

**Used By:**

- `internal/domain/entity/models.go` — registered after `Calendar`

**FK Constraint:** `calendar_id → calendars.id` (required, not nullable)

---

### `internal/domain/entity/schedule.go` (MODIFIED)

**Purpose:** Links a `DecisionRule` to a `Placement` with a recurrence-based time window. Refactored from simple `StartTimestamp`/`EndTimestamp` to a full cron architecture.

**Lines of Code:** ~36 (was ~20 before refactor)

**What Future Contributors Must Know:**

- `CalendarID` is **nullable** (`*uuid.UUID`) — only populated when `RecurrenceType = CALENDAR`
- `RecurrenceRule` is **nullable** (`*string`) — only populated when `RecurrenceType = RRULE`; contains raw RFC 5545 RRULE string (e.g., `FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR`)
- `TimeOfDayStart`/`TimeOfDayEnd` are **nullable** (`*string`) — use `"HH:MM"` format, `nil` when `AllDay = true`
- The composite unique index `idx_rule_placement` on `(decision_rule_id, placement_id)` is preserved
- `Timezone` defaults to `'Asia/Bangkok'` — system is designed as single-timezone per brainstorm constraint
- **Breaking change:** `start_timestamp` and `end_timestamp` columns removed from DB

**Exports:**

| Symbol     | Kind   | Description                    |
| ---------- | ------ | ------------------------------ |
| `Schedule` | struct | GORM model; table: `schedules` |

**Fields:**

| Field            | DB Column           | Type            | GORM Tag                         | JSON                     | Nullable           |
| ---------------- | ------------------- | --------------- | -------------------------------- | ------------------------ | ------------------ |
| `BaseModel`      | —                   | —               | UUID PK, audit                   | —                        | —                  |
| `DecisionRuleID` | `decision_rule_id`  | `uuid`          | `uniqueIndex:idx_rule_placement` | `decisionRuleId`         | No                 |
| `DecisionRule`   | —                   | `*DecisionRule` | `foreignKey:DecisionRuleID`      | `decisionRule,omitempty` | —                  |
| `PlacementID`    | `placement_id`      | `uuid`          | `uniqueIndex:idx_rule_placement` | `placementId`            | No                 |
| `Placement`      | —                   | `*Placement`    | `foreignKey:PlacementID`         | `placement,omitempty`    | —                  |
| `CalendarID`     | `calendar_id`       | `uuid`          | `type:uuid`                      | `calendarId`             | Yes (`*uuid.UUID`) |
| `Calendar`       | —                   | `*Calendar`     | `foreignKey:CalendarID`          | `calendar,omitempty`     | —                  |
| `RecurrenceType` | `recurrence_type`   | `varchar(255)`  | `size:255`                       | `recurrenceType`         | No                 |
| `RecurrenceRule` | `recurrence_rule`   | `text`          | `type:text`                      | `recurrenceRule`         | Yes (`*string`)    |
| `EffectiveFrom`  | `effective_from`    | `timestamptz`   | `type:timestamptz`               | `effectiveFrom`          | No                 |
| `EffectiveUntil` | `effective_until`   | `timestamptz`   | `type:timestamptz`               | `effectiveUntil`         | No                 |
| `TimeOfDayStart` | `time_of_day_start` | `varchar(5)`    | `size:5`                         | `timeOfDayStart`         | Yes (`*string`)    |
| `TimeOfDayEnd`   | `time_of_day_end`   | `varchar(5)`    | `size:5`                         | `timeOfDayEnd`           | Yes (`*string`)    |
| `AllDay`         | `all_day`           | `boolean`       | `default:false`                  | `allDay`                 | No                 |
| `Timezone`       | `timezone`          | `varchar(255)`  | `default:'Asia/Bangkok'`         | `timezone`               | No                 |
| `IsActive`       | `is_active`         | `boolean`       | `default:false`                  | `isActive`               | No                 |

**Dependencies:**

- `time` — `EffectiveFrom`, `EffectiveUntil`
- `kbank-ecms/internal/domain/entity/enums` — `RecurrenceType`
- `github.com/google/uuid`

---

### `internal/domain/entity/schedule_occurrence.go`

**Purpose:** Stores a materialized instance of a Schedule. Acts as the append-only event log that future cron jobs write to and the FullCalendar API reads from.

**Lines of Code:** ~25

**What Future Contributors Must Know:**

- Records are generated by a cron job (future story) — this entity is **read-mostly** after creation
- `OccurrenceStart`/`OccurrenceEnd` can differ from template `EffectiveFrom`/`EffectiveUntil` when `Status = MODIFIED`
- Do not delete occurrences — use `Status = CANCELLED` instead for audit trail integrity
- `Source` is write-once metadata for traceability

**Exports:**

| Symbol               | Kind   | Description                               |
| -------------------- | ------ | ----------------------------------------- |
| `ScheduleOccurrence` | struct | GORM model; table: `schedule_occurrences` |

**Fields:**

| Field             | DB Column          | Type            | GORM Tag                | JSON                 | Nullable |
| ----------------- | ------------------ | --------------- | ----------------------- | -------------------- | -------- |
| `BaseModel`       | —                  | —               | UUID PK, audit          | —                    | —        |
| `ScheduleID`      | `schedule_id`      | `uuid NOT NULL` | `type:uuid;not null`    | `scheduleId`         | No       |
| `Schedule`        | —                  | `*Schedule`     | `foreignKey:ScheduleID` | `schedule,omitempty` | —        |
| `OccurrenceStart` | `occurrence_start` | `timestamptz`   | `type:timestamptz`      | `occurrenceStart`    | No       |
| `OccurrenceEnd`   | `occurrence_end`   | `timestamptz`   | `type:timestamptz`      | `occurrenceEnd`      | No       |
| `Status`          | `status`           | `varchar(255)`  | `size:255`              | `status`             | No       |
| `Source`          | `source`           | `varchar(255)`  | `size:255`              | `source`             | No       |

**Dependencies:**

- `time`
- `kbank-ecms/internal/domain/entity/enums` — `OccurrenceStatus`, `OccurrenceSource`
- `github.com/google/uuid`

**Used By:**

- `internal/domain/entity/models.go` — registered after `Schedule`

---

### `internal/domain/entity/models.go` (MODIFIED)

**Purpose:** Defines GORM `AutoMigrate` order for all entities. Updated to include 3 new tables in correct FK dependency order.

**What Future Contributors Must Know:**  
Order matters. `Calendar` must come before `CalendarDate` and `Schedule`. `ScheduleOccurrence` must come after `Schedule`. Violating this order will cause FK constraint errors during DB migration.

**Migration Order (updated):**

```
Role, Profile, Placement, MDPSchemaRegistry, Calendar          ← independent
User, Permission, Attribute, LoginTokenHistory, CalendarDate   ← FK to independent
ProfilePermission, DecisionRule, Rule, RuleCondition           ← junction/dependent
Schedule, ScheduleOccurrence                                   ← scheduling chain
```

---

## Dependency Graph

```
enums/calendar_type.go
    └─▶ calendar.go
            └─▶ calendar_date.go
            └─▶ schedule.go (nullable FK)

enums/recurrence_type.go
    └─▶ schedule.go

enums/occurrence_status.go  ┐
enums/occurrence_source.go  ┤─▶ schedule_occurrence.go
    schedule.go             ┘       (FK → schedule)

models.go ─▶ registers all 4 new entities
```

---

## Data Flow Analysis

### Use Case → Schema Mapping

| Use Case                          | `recurrence_type` | `recurrence_rule`                     | `calendar_id`         | `all_day` | `time_of_day_start/end` |
| --------------------------------- | ----------------- | ------------------------------------- | --------------------- | --------- | ----------------------- |
| Banner Mon–Fri 09:00–17:00 in May | `RRULE`           | `FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR`    | `null`                | `false`   | `"09:00"` / `"17:00"`   |
| Campaign 1–31 May (24 hr)         | `ONCE`            | `null`                                | `null`                | `true`    | `null` / `null`         |
| Public Holidays                   | `CALENDAR`        | `null`                                | → holiday calendar ID | `true`    | `null` / `null`         |
| Birthday June 15 yearly           | `RRULE`           | `FREQ=YEARLY;BYMONTH=6;BYMONTHDAY=15` | `null`                | `true`    | `null` / `null`         |

### Occurrence Generation Flow (Future Cron — Out of Scope)

```
Schedule (template)
    │ recurrence_type = RRULE  → parse recurrence_rule → expand dates
    │ recurrence_type = CALENDAR → query calendar_dates WHERE calendar_id = ? → expand dates
    │ recurrence_type = ONCE    → create single occurrence from effective_from/until
    ▼
ScheduleOccurrence (materialized)
    source = RECURRENCE | CALENDAR | MANUAL
    status = ACTIVE (initial)
```

---

## Integration Points

| Integration             | Direction     | Details                                    |
| ----------------------- | ------------- | ------------------------------------------ |
| `decision_rules` table  | FK (required) | Schedule belongs to one DecisionRule       |
| `placements` table      | FK (required) | Schedule belongs to one Placement          |
| `calendars` table       | FK (nullable) | Only when `recurrence_type = CALENDAR`     |
| `schedule_occurrences`  | One-to-many   | Schedule has many occurrences              |
| Future cron job service | Consumer      | Reads Schedule, writes ScheduleOccurrences |
| Future FullCalendar API | Consumer      | Reads ScheduleOccurrences as events        |

---

## Related Code & Patterns

### Enum Pattern Reference

All 4 new enums follow `internal/domain/entity/enums/decision_rule_status.go` exactly:

- Type alias (`type Name string`)
- Grouped constants
- `String()`, `IsValid()`, `Parse*()` methods

### FK Pattern Reference

- **Required FK:** `uuid.UUID` + `gorm:"type:uuid;not null"` → see `CalendarDate.CalendarID`
- **Nullable FK:** `*uuid.UUID` + `gorm:"type:uuid"` → see `Schedule.CalendarID`
- **Relationship field:** always `*RelatedEntity` with `json:"...,omitempty"`

### Comparable Entities

- `ProfilePermission` — unique composite index pattern (same as `Schedule.idx_rule_placement`)
- `RuleCondition` — nullable FK pattern (`ParentRuleConditionID *uuid.UUID`)

---

## Contributor Checklist

### Risks & Gotchas

1. **`start_timestamp`/`end_timestamp` removed** — any existing DB migrations, seed scripts, or API DTOs referencing these columns will break
2. **`TimeOfDayStart`/`TimeOfDayEnd` are nullable `*string`** — always nil-check before parsing as time; do not default to `""` (empty string is invalid HH:MM)
3. **`RecurrenceRule` is nullable** — writing `FREQ=...` when `RecurrenceType = ONCE` is logically invalid; enforce this in service layer, not entity layer
4. **`CalendarID` nullable** — querying Schedule with `Preload("Calendar")` when `CalendarID = null` will result in `Calendar = nil`, handle accordingly
5. **GORM `type:date`** — `CalendarDate.Date` saves as PostgreSQL `date`; Go zero value (`0001-01-01`) will be stored if not explicitly set
6. **Migration order** — if you add new entities with FK to `calendars` or `schedule_occurrences`, update `AllModels()` in correct order

### Pre-change Verification Steps

```bash
# Verify no compilation errors
go build ./...

# Verify enum tests still pass
go test -v ./internal/domain/entity/enums/

# Verify no regressions in full suite
go test ./...
```

### Suggested Tests Before PR

- Unit test any new `ParseXxx()` calls at API boundary
- Test nil-safety for `CalendarID`, `RecurrenceRule`, `TimeOfDayStart`, `TimeOfDayEnd`
- Add integration test for `AllModels()` AutoMigrate order (requires test DB)
- Test `IsRecurring = true` date expansion logic in cron service (future story)

---

## Out of Scope (Future Stories)

| Feature                        | Details                                                              |
| ------------------------------ | -------------------------------------------------------------------- |
| Occurrence generation cron job | Reads `Schedule`, writes `ScheduleOccurrence` using RRULE parser     |
| RRULE parsing                  | `teambition/rrule-go` or similar RFC 5545 library                    |
| FullCalendar event API         | `GET /api/schedules/events?start=&end=` returning occurrence windows |
| Calendar CRUD admin API        | Endpoints for managing `Calendar` and `CalendarDate` records         |
| Conflict resolution            | Detect overlapping occurrences for same `placement_id`               |
| Frontend integration           | FullCalendar display component                                       |
