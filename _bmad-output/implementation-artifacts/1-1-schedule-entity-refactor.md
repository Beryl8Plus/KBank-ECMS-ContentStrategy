# Story 1.1: Schedule Entity Refactor — Three-Table Cron Architecture

Status: review

## Story

As a **backend developer**,
I want to **refactor the Schedule entity from simple start/end timestamps to a three-table architecture (schedules, schedule_occurrences, calendars/calendar_dates) supporting RRULE-based recurrence, calendar-linked holidays, and one-time date ranges**,
so that **the system can represent all four scheduling patterns (weekday banner, date-range campaign, public holidays, personal birthdays) and expose a FullCalendar-compatible event API for the frontend**.

## Acceptance Criteria

1. **AC-1: Schedule entity refactored with recurrence support**
   - Given a Schedule entity
   - When it is created with `recurrence_type = ONCE | RRULE | CALENDAR`
   - Then it stores `recurrence_rule` (RFC 5545 RRULE string, nullable), `calendar_id` (FK, nullable), `effective_from`/`effective_until` (timestamptz), `time_of_day_start`/`time_of_day_end` (string in "HH:MM" format), `all_day` (bool), `timezone` (text, default "Asia/Bangkok"), and `is_active` (bool)
   - And the composite unique index on `(decision_rule_id, placement_id)` is retained

2. **AC-2: ScheduleOccurrence entity stores materialized instances**
   - Given a Schedule with any recurrence type
   - When occurrences are generated (manually or by future cron job)
   - Then each occurrence is stored with `schedule_id` (FK), `occurrence_start`/`occurrence_end` (timestamptz), `status` (ACTIVE | CANCELLED | MODIFIED), and `source` (RECURRENCE | CALENDAR | MANUAL)

3. **AC-3: Calendar and CalendarDate entities for holiday/personal sources**
   - Given a Calendar entity with `name`, `type` (HOLIDAY | PERSONAL | CUSTOM), and `is_active`
   - When CalendarDate entries are added
   - Then each stores `calendar_id` (FK), `date` (date type), `name` (text), `is_recurring` (bool)

4. **AC-4: GORM auto-migration succeeds in correct dependency order**
   - Given the updated `AllModels()` function
   - When `AutoMigrate` runs
   - Then Calendar → CalendarDate → Schedule (refactored) → ScheduleOccurrence tables are created without FK errors

5. **AC-5: New enums follow existing project conventions**
   - Given new enum types `RecurrenceType`, `OccurrenceStatus`, `OccurrenceSource`, `CalendarType`
   - When defined in `internal/domain/entity/enums/`
   - Then each includes `String()`, `IsValid()`, `Parse*()` functions matching the pattern in `decision_rule_status.go`

6. **AC-6: All four use cases are representable**
   - Given the new schema
   - When creating schedules for: (a) Banner Mon-Fri 9-17 in May, (b) Campaign 1-31 May 24hr, (c) Public Holidays via calendar, (d) Birthday June 15 yearly
   - Then each maps correctly to the schema as defined in the brainstorm use-case table

## Tasks / Subtasks

- [x] **Task 1: Create new enum types** (AC: #5)
  - [x] 1.1 Create `internal/domain/entity/enums/recurrence_type.go` — `ONCE | RRULE | CALENDAR`
  - [x] 1.2 Create `internal/domain/entity/enums/occurrence_status.go` — `ACTIVE | CANCELLED | MODIFIED`
  - [x] 1.3 Create `internal/domain/entity/enums/occurrence_source.go` — `RECURRENCE | CALENDAR | MANUAL`
  - [x] 1.4 Create `internal/domain/entity/enums/calendar_type.go` — `HOLIDAY | PERSONAL | CUSTOM`

- [x] **Task 2: Create Calendar and CalendarDate entities** (AC: #3, #4)
  - [x] 2.1 Create `internal/domain/entity/calendar.go` — Calendar struct with BaseModel, Name, Type, IsActive
  - [x] 2.2 Create `internal/domain/entity/calendar_date.go` — CalendarDate struct with BaseModel, CalendarID (FK), Date, Name, IsRecurring

- [x] **Task 3: Refactor Schedule entity** (AC: #1)
  - [x] 3.1 Replace `StartTimestamp`/`EndTimestamp` with new fields in `internal/domain/entity/schedule.go`
  - [x] 3.2 Add `CalendarID` nullable FK, `RecurrenceType` enum, `RecurrenceRule` text, `EffectiveFrom`/`EffectiveUntil` timestamptz, `TimeOfDayStart`/`TimeOfDayEnd` string, `AllDay` bool, `Timezone` text

- [x] **Task 4: Create ScheduleOccurrence entity** (AC: #2)
  - [x] 4.1 Create `internal/domain/entity/schedule_occurrence.go` — ScheduleOccurrence struct with BaseModel, ScheduleID (FK), OccurrenceStart/End, Status, Source

- [x] **Task 5: Update auto-migration order** (AC: #4)
  - [x] 5.1 Update `internal/domain/entity/models.go` — add Calendar, CalendarDate before Schedule; add ScheduleOccurrence after Schedule

- [x] **Task 6: Verify compilation and migration** (AC: #4, #6)
  - [x] 6.1 Run `go build ./...` to verify no compilation errors
  - [x] 6.2 Verify all four use-case patterns are correctly representable

## Dev Notes

### Source Reference

- **Brainstorm session:** `_bmad-output/brainstorming/brainstorming-session-2026-04-07-01.md`
- **Recommended Architecture:** Section "Recommended Architecture" in brainstorm — 3-table design
- **Use Case Mapping:** Section "Use Case → Schema Mapping" table in brainstorm
- **API spec (current):** `docs/api/decision-rule-management-apis/save-decision-rule-step-3.md`

### Project Conventions (MUST FOLLOW)

**Entity pattern** — follow exact style from existing entities:
```go
// Comment describing entity purpose.
//
// Table: table_name_plural
type EntityName struct {
    BaseModel
    FieldName Type `gorm:"..." json:"..."`
}
```

**Enum pattern** — follow `decision_rule_status.go` exactly:
- Type alias: `type EnumName string`
- Constants: `const ( EnumNameValue1 EnumName = "VALUE1" ... )`
- Methods: `String()`, `IsValid()`, `ParseEnumName(s string) (EnumName, error)`

**GORM tag alignment** — all tag columns are right-aligned with spaces (see existing entities)

**JSON naming** — camelCase in json tags

**FK pattern** — `FieldID uuid.UUID` + `Field *RelatedEntity` with `gorm:"foreignKey:FieldID"`

**Nullable FK** — use `*uuid.UUID` for nullable foreign keys (see `User.RoleID` pattern)

### Schema Design (from brainstorm session)

```
schedules (Template/Pattern) — REFACTORED
├── id (uuid PK, from BaseModel)
├── decision_rule_id (uuid FK → decision_rules, unique with placement_id)
├── placement_id (uuid FK → placements, unique with decision_rule_id)
├── calendar_id (nullable uuid FK → calendars)
├── recurrence_type (varchar — ONCE | RRULE | CALENDAR)
├── recurrence_rule (text, nullable — RFC 5545 RRULE string)
├── effective_from (timestamptz)
├── effective_until (timestamptz)
├── time_of_day_start (varchar — "HH:MM" format, Go string)
├── time_of_day_end (varchar — "HH:MM" format, Go string)
├── all_day (bool, default false)
├── timezone (varchar, default "Asia/Bangkok")
├── is_active (bool, default false)
└── BaseModel audit fields

schedule_occurrences (Materialized Instances) — NEW
├── id (uuid PK, from BaseModel)
├── schedule_id (uuid FK → schedules)
├── occurrence_start (timestamptz)
├── occurrence_end (timestamptz)
├── status (varchar — ACTIVE | CANCELLED | MODIFIED)
├── source (varchar — RECURRENCE | CALENDAR | MANUAL)
└── BaseModel audit fields

calendars (Holiday/Personal Sources) — NEW
├── id (uuid PK, from BaseModel)
├── name (varchar 255)
├── type (varchar — HOLIDAY | PERSONAL | CUSTOM)
├── is_active (bool, default true)
└── BaseModel audit fields

calendar_dates — NEW
├── id (uuid PK, from BaseModel)
├── calendar_id (uuid FK → calendars)
├── date (date — PostgreSQL native date type)
├── name (varchar 255)
├── is_recurring (bool, default false)
└── BaseModel audit fields
```

### Use Case → Schema Mapping (validation reference)

| Use Case | recurrence_type | recurrence_rule | calendar_id | all_day | time_of_day |
|----------|----------------|----------------|-------------|---------|-------------|
| Banner Mon-Fri 9-17 May | RRULE | `FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR` | null | false | 09:00-17:00 |
| Campaign 1-31 May 24hr | ONCE | null | null | true | null |
| Public Holidays | CALENDAR | null | → holiday_cal | true | null |
| Birthday June 15 yearly | RRULE | `FREQ=YEARLY;BYMONTH=6;BYMONTHDAY=15` | null | true | null |

### Critical Design Decisions

1. **TimeOfDay as string "HH:MM"** — Go lacks native time-of-day type; PostgreSQL `time` maps poorly to Go. Store as `varchar` with "HH:MM" format. Parse in service layer when needed.
2. **Timezone field** — default `"Asia/Bangkok"`. System is single-timezone per brainstorm constraint #39.
3. **Unique index preserved** — `idx_rule_placement` on `(decision_rule_id, placement_id)` remains.
4. **Calendar FK is nullable** — only populated when `recurrence_type = CALENDAR`.
5. **RecurrenceRule is nullable** — only populated when `recurrence_type = RRULE`.
6. **This story is ENTITY-ONLY** — no service logic, no cron job, no API endpoint. Those are separate stories.

### Files to Create/Modify

| Action | File Path |
|--------|-----------|
| CREATE | `internal/domain/entity/enums/recurrence_type.go` |
| CREATE | `internal/domain/entity/enums/occurrence_status.go` |
| CREATE | `internal/domain/entity/enums/occurrence_source.go` |
| CREATE | `internal/domain/entity/enums/calendar_type.go` |
| CREATE | `internal/domain/entity/calendar.go` |
| CREATE | `internal/domain/entity/calendar_date.go` |
| MODIFY | `internal/domain/entity/schedule.go` |
| CREATE | `internal/domain/entity/schedule_occurrence.go` |
| MODIFY | `internal/domain/entity/models.go` |

### Out of Scope (future stories)

- Occurrence generation service / cron job
- RRULE parsing with `teambition/rrule-go`
- FullCalendar event API endpoint (`GET /api/schedules/events?start=&end=`)
- Calendar CRUD admin API
- Conflict resolution / overlap detection
- PostgreSQL `tstzrange` constraints
- Frontend integration

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (GitHub Copilot)

### Debug Log References

None — all tasks completed without errors.

### Completion Notes List

- All 4 enum types created with `String()`, `IsValid()`, `Parse*()` methods matching `decision_rule_status.go` convention (AC-5)
- Calendar entity created with BaseModel, Name, Type (CalendarType enum), IsActive fields (AC-3)
- CalendarDate entity created with CalendarID FK to Calendar, Date (date type), Name, IsRecurring (AC-3)
- Schedule entity refactored: removed StartTimestamp/EndTimestamp, added RecurrenceType, RecurrenceRule (*string nullable), CalendarID (*uuid.UUID nullable FK), EffectiveFrom/Until (timestamptz), TimeOfDayStart/End (*string), AllDay, Timezone (default Asia/Bangkok), IsActive. Composite unique index on (decision_rule_id, placement_id) retained (AC-1)
- ScheduleOccurrence entity created with ScheduleID FK, OccurrenceStart/End (timestamptz), Status (OccurrenceStatus), Source (OccurrenceSource) (AC-2)
- AllModels() updated with correct dependency order: Calendar → CalendarDate → Schedule → ScheduleOccurrence (AC-4)
- `go build ./...` passes with zero errors (AC-4)
- All 52 enum unit tests pass (4 test files covering IsValid, Parse*, String for each enum)
- All 7 base_model tests pass — no regressions
- All 4 use cases verified representable against schema (AC-6)

### File List

| Action | File |
|--------|------|
| CREATE | `internal/domain/entity/enums/recurrence_type.go` |
| CREATE | `internal/domain/entity/enums/recurrence_type_test.go` |
| CREATE | `internal/domain/entity/enums/occurrence_status.go` |
| CREATE | `internal/domain/entity/enums/occurrence_status_test.go` |
| CREATE | `internal/domain/entity/enums/occurrence_source.go` |
| CREATE | `internal/domain/entity/enums/occurrence_source_test.go` |
| CREATE | `internal/domain/entity/enums/calendar_type.go` |
| CREATE | `internal/domain/entity/enums/calendar_type_test.go` |
| CREATE | `internal/domain/entity/calendar.go` |
| CREATE | `internal/domain/entity/calendar_date.go` |
| MODIFY | `internal/domain/entity/schedule.go` |
| CREATE | `internal/domain/entity/schedule_occurrence.go` |
| MODIFY | `internal/domain/entity/models.go` |

### Change Log

- **2026-04-07:** Story 1.1 implemented — Schedule entity refactored to three-table cron architecture with 4 new enums, 3 new entities, 1 refactored entity, migration order updated, 52 enum tests + 7 base model tests all passing.
