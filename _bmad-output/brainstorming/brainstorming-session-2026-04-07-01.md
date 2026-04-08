---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: []
session_topic: 'Refactor Schedule entity time model from simple start/end timestamps to scalable cron-based recurring scheduling with frontend calendar support'
session_goals: 'Explore new data model designs, recurring schedule representations (cron, RRULE, etc.), and API design patterns for frontend calendar integration'
selected_approach: 'ai-recommended'
techniques_used: ['Morphological Analysis', 'Analogical Thinking', 'Constraint Mapping + Reverse Brainstorming']
ideas_generated: [44]
context_file: ''
session_active: false
workflow_completed: true
---

# Brainstorming Session Results

**Facilitator:** Nrtdemo
**Date:** 2026-04-07

## Session Overview

**Topic:** Refactor Schedule entity time model — evolve from simple `StartTimestamp`/`EndTimestamp` pair to a scalable structure supporting cron-based recurring scheduling and frontend calendar integration.

**Goals:**
- Explore new data model designs that go beyond a single time-window pair
- Evaluate recurring schedule representations (cron expression, RRULE, custom patterns)
- Design API contracts that frontend calendar components can consume easily
- Consider migration path from current simple model to new structure

### Current State (Context)

```go
// Schedule links a DecisionRule to a Placement with an active time window.
type Schedule struct {
    BaseModel
    DecisionRuleID uuid.UUID     `gorm:"type:uuid;uniqueIndex:idx_rule_placement"`
    PlacementID    uuid.UUID     `gorm:"type:uuid;uniqueIndex:idx_rule_placement"`
    StartTimestamp time.Time     `gorm:"type:timestamptz"`
    EndTimestamp   time.Time     `gorm:"type:timestamptz"`
    IsActive       bool          `gorm:"default:false"`
}
```

**Key Constraints:** Schedule currently ties a DecisionRule to a Placement with a single active time window. No recurring/repeating capability exists.

## Technique Selection

**Approach:** AI-Recommended Techniques
**Analysis Context:** Refactoring Schedule time model with focus on data model, recurrence representation, and calendar API design

**Recommended Techniques:**
- **Morphological Analysis:** Decompose all independent dimensions (time representation, granularity, recurrence scope, API format) and explore all combinations systematically
- **Analogical Thinking:** Draw patterns from proven systems (Google Calendar, crontab, Airflow, K8s CronJobs, iCalendar RFC 5545) and map to our use case
- **Constraint Mapping + Reverse Brainstorming:** Map all real constraints (GORM, PostgreSQL, migration, frontend calendar libs) and stress-test designs by asking "what would break this?"

**AI Rationale:** Topic requires structured decomposition first, then creative pattern borrowing, then reality-check filtering — progressing from broad exploration to battle-tested shortlist.

## Technique Execution Results

### Phase 1: Morphological Analysis

**Interactive Focus:** Decomposed 4 independent dimensions of the scheduling problem
**User Creative Contributions:** Provided 4 real-world use cases that revealed fundamentally different scheduling patterns

**Key Ideas Generated:**

- **[TimeModel #1]** Hybrid Window + Recurrence Pattern — separate EffectiveFrom/Until from RecurrenceType + RecurrenceConfig
- **[TimeModel #2]** Multi-Layer Parent-Child — parent schedule pattern → child occurrence records
- **[TimeModel #3]** Schedule + Calendar Source — external calendar reference for holidays
- **[TimeModel #4]** Holiday Calendar Entity + Multi-Source Input — admin manual + preset + CSV import
- **[TimeModel #5]** FullCalendar-Native Event Model — API response matching FullCalendar Event Object
- **[TimeModel #6]** Materialized Occurrences + Lazy Generation — precompute N months ahead
- **[DataArch #7]** Calendar-Linked Schedule — nullable FK, unified occurrence query
- **[DataArch #8]** Occurrence Override Pattern — is_override flag for exceptions
- **[DataArch #9]** FullCalendar Event Feed API — GET /schedules/events?start=&end=
- **[DataArch #10]** Lightweight Materialization — truncate + regenerate at ~100 scale
- **[DataArch #11]** Override as First-Class Citizen — status enum: active/cancelled/modified
- **[DataArch #12]** Admin Holiday Calendar CRUD — simple entity, no external API
- **[API #13]** Pure Occurrence Feed — backend-expanded flat event list
- **[API #14]** Grouped Event Response — FullCalendar groupId for bulk operations
- **[API #15]** Calendar View Optimized Response — summary for month, detail for day
- **[API #16]** Event Action Endpoints — cancel/reschedule/regenerate

**Morphological Matrix:**

| Dimension | Options Explored |
|-----------|-----------------|
| Time Representation | Hybrid Window+Recurrence, RRULE (RFC 5545), Materialized Occurrences |
| Granularity | Multi-level (minute → yearly), AllDay flag, DaysOfWeek |
| Data Architecture | 3-table design, Lightweight materialization, Override first-class |
| API Contract | Pure Occurrence Feed, Grouped response, View-optimized |

### Phase 2: Analogical Thinking

**Building on Phase 1:** Drew patterns from 5 real-world systems and mapped to our use case
**Analogies Explored:** Google Calendar, Kubernetes CronJobs, Apache Airflow, FullCalendar Internal Model, Banking Campaign Systems

**Key Ideas Generated:**

- **[Analogy #17]** Google-Style Exception Model — exception instance with recurringEventId trace
- **[Analogy #18]** Calendar Layer Overlay — toggle-able calendar layers per source
- **[Analogy #19]** RRULE as Lingua Franca — RFC 5545 standard across backend + frontend
- **[Analogy #20]** ConcurrencyPolicy for Schedules — allow/priority/reject on placement overlap
- **[Analogy #21]** Schedule-as-Template Pattern — template → instances, edit propagates to future only
- **[Analogy #22]** Deadline-Aware Generation — backfill policy on cron job recovery
- **[Analogy #23]** Interval-Based Recurrence — "every 4 hours" without cron
- **[Analogy #24]** Data Window Per Occurrence — explicit window_start/end per occurrence
- **[Analogy #25]** Multi-Source Event Feed — separate endpoints per calendar type
- **[Analogy #26]** Backend-as-FullCalendar-Adapter — anti-corruption layer for frontend lib
- **[Analogy #27]** Lazy Fetch + Precomputed Range — FullCalendar lazy fetching + materialized
- **[Analogy #28]** Campaign Priority + Conflict Resolution — DecisionRule.Score-based priority
- **[Analogy #29]** Blackout Periods — negative rules overriding positive schedules
- **[Analogy #30]** Approval Workflow for Schedule Changes — pending approval before apply

### Phase 3: Constraint Mapping + Reverse Brainstorming

**Constraints Mapped:** GORM/Go, PostgreSQL, Migration, FullCalendar frontend
**Risks Identified:** Timezone, stale occurrences, job resilience, collision resolution

**Key Ideas Generated:**

- **[Constraint #31]** TimeOfDay Representation — Go lacks native time-only type
- **[Constraint #32]** RRULE Go Library Assessment — teambition/rrule-go verification needed
- **[Constraint #33]** PostgreSQL Range Type (tstzrange) — DB-level overlap detection
- **[Constraint #34]** DB-Side Occurrence Generation — generate_series() in SQL
- **[Constraint #35]** Greenfield Advantage — no production data, optimal redesign time
- **[Constraint #36]** Unique Index Reinterpretation — idx_rule_placement needs rethinking
- **[Constraint #37]** Timezone Alignment Strategy — UTC storage, ISO 8601 API, Asia/Bangkok frontend
- **[Constraint #38]** Server-Side Expansion Default — don't rely on frontend rrule plugin
- **[Risk #39]** Multi-Timezone Admin — force Asia/Bangkok as single-timezone system
- **[Risk #40]** Occurrence Invalidation — preserve overrides on pattern change
- **[Risk #41]** Calendar Change Propagation — trigger regeneration on calendar update
- **[Risk #42]** Generation Job Resilience — dual-mode: materialized + on-the-fly fallback
- **[Risk #43]** Query Performance — index strategy + optional month-view aggregation
- **[Risk #44]** Schedule Collision Resolution — business decision: allow/priority/block overlaps

### Creative Facilitation Narrative

Session began with systematic decomposition of the scheduling problem into 4 independent dimensions. User's 4 real-world use cases (weekday banner, date-range campaign, public holidays, personal anniversaries) became the cornerstone — revealing that no single representation covers all patterns. Cross-pollination from Google Calendar, K8s CronJobs, and Airflow inspired the template-instance pattern and resilience mechanisms. Constraint mapping confirmed PostgreSQL's strong capabilities (tstzrange, generate_series) and the greenfield advantage. Reverse brainstorming exposed timezone and stale-occurrence risks early.

## Idea Organization and Prioritization

### Thematic Organization

**Theme 1: Data Model Architecture** — 8 ideas → converge on 3-table design (schedules, schedule_occurrences, calendars/calendar_dates)
**Theme 2: Recurrence Representation** — 5 ideas → RRULE (RFC 5545) as primary, with Go library assessment required
**Theme 3: Calendar & Holiday Management** — 4 ideas → admin-managed calendar entity with multi-source input
**Theme 4: API Design & FullCalendar** — 8 ideas → server-side expanded occurrence feed with FullCalendar adapter layer
**Theme 5: Conflict Resolution & Business Rules** — 8 ideas → override status enum + priority-based conflict resolution
**Theme 6: Resilience & Operations** — 10 ideas → timezone contract, smart regeneration, dual-mode fallback

### Prioritization Results

**Top Priority (v1 Must-Have):**

1. **Three-Table Architecture** — schedules (template) + schedule_occurrences (materialized) + calendars/calendar_dates
2. **Server-Side Occurrence Expansion + FullCalendar Adapter** — backend expand all recurrences, return flat FullCalendar JSON
3. **Timezone Contract + Override Model** — fixed Asia/Bangkok timezone, occurrence status enum, preserve overrides on regeneration

**Quick Wins:**

- Greenfield advantage — redesign schema now with zero migration cost
- Admin Calendar CRUD — simple entity, fast to build
- Unique index drop + recreate in new migration

**Breakthrough Concepts (v2 Roadmap):**

- ConcurrencyPolicy — priority-based schedule conflict resolution
- Blackout Periods — negative scheduling rules
- Dual-Mode resilience — materialized + on-the-fly fallback
- PostgreSQL tstzrange — DB-enforced overlap prevention

### Action Planning

**Action Plan 1: Three-Table Schema Design**

- **Immediate:** Create Go entity files for Schedule (refactored), ScheduleOccurrence, Calendar, CalendarDate
- **Resources:** GORM migration, PostgreSQL time/timestamptz types
- **Timeline:** Entity + migration in first sprint
- **Success:** All 4 use cases representable in schema

**Action Plan 2: Occurrence Generation Service**

- **Immediate:** Assess `teambition/rrule-go` library for RRULE parsing/expansion
- **Resources:** Go service layer, PostgreSQL generate_series for CALENDAR type
- **Timeline:** Service + cron job in second sprint
- **Success:** Occurrences auto-generated, overrides preserved on regeneration

**Action Plan 3: FullCalendar Event API**

- **Immediate:** Define API contract: `GET /api/schedules/events?start=&end=`
- **Resources:** FullCalendar Event Object spec, HTTP handler + transformer layer
- **Timeline:** API endpoint in second sprint
- **Success:** FullCalendar renders events with zero frontend transformation

### Recommended Architecture

```
schedules (Template/Pattern)
├── id, decision_rule_id, placement_id
├── calendar_id (nullable FK → calendars)
├── recurrence_type: ONCE | RRULE | CALENDAR
├── recurrence_rule: text (RFC 5545 RRULE string)
├── effective_from/until: timestamptz
├── time_of_day_start/end: time (PostgreSQL native)
├── all_day: bool, timezone: text, is_active: bool
│
├──► schedule_occurrences (Materialized Instances)
│    ├── schedule_id FK, occurrence_start/end: timestamptz
│    ├── status: ACTIVE | CANCELLED | MODIFIED
│    └── source: RECURRENCE | CALENDAR | MANUAL
│
└──► calendars (Holiday/Personal Sources)
     ├── name, type: HOLIDAY | PERSONAL | CUSTOM
     └──► calendar_dates (date, name)
```

**Use Case → Schema Mapping:**

| Use Case | recurrence_type | recurrence_rule | calendar_id | all_day |
|----------|----------------|----------------|-------------|---------|
| Banner Mon-Fri 9-17 May | RRULE | FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR | null | false |
| Campaign 1-31 May 24hr | ONCE | null | null | true |
| Public Holidays | CALENDAR | null | → holiday_cal | true |
| Birthday June 15 | RRULE | FREQ=YEARLY;BYMONTH=6;BYMONTHDAY=15 | null | true |

## Session Summary and Insights

**Key Achievements:**

- Generated 44 ideas across 6 themes using 3 complementary techniques
- Synthesized a concrete 3-table architecture design from divergent exploration
- Identified critical risks (timezone, stale occurrences) early in design phase
- Mapped all 4 user use cases to the recommended schema
- Created clear v1/v2 roadmap with actionable next steps

**Session Reflections:**

The combination of Morphological Analysis (systematic decomposition) → Analogical Thinking (proven patterns) → Constraint Mapping + Reverse Brainstorming (reality check) proved highly effective for architectural design decisions. The 4 real-world use cases provided by the user became the validation backbone throughout all phases — every idea was tested against these concrete scenarios.
