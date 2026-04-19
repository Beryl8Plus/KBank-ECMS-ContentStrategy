# Story 2.1: CMS Runtime & Delivery — Domain Service Interfaces

Status: done

## Story

As a **backend developer**,
I want to **define the `RuntimeService` and `DeliveryService` Go interfaces in `internal/domain/service/`, along with the `RuleEvaluator` strategy interface and `Registry` in `internal/service/evaluator/`**,
so that **the cms-runtime and cms-delivery-service implementations are decoupled from infrastructure details, enforce a clean contract for future microservice extraction, and enable the strategy pattern for rule evaluation**.

## Acceptance Criteria

1. **AC-1: RuntimeService interface defined in domain/service**
   - Given a new file `internal/domain/service/cms_runtime.go`
   - When it is compiled
   - Then it exposes: `EvaluateAll(ctx context.Context) error`, `EvaluatePlacement(ctx context.Context, placementName string) error`, `Start(ctx context.Context) error`, `Stop() error`
   - And the interface is in package `service`

2. **AC-2: ContentResult and DeliveryService interface defined in domain/service**
   - Given a new file `internal/domain/service/cms_delivery.go`
   - When it is compiled
   - Then it defines `ContentResult` struct with fields: `ContentPath string`, `AemURL string`, `Score float64`, `RuleID string`, `RuleType string`, `EvaluatedAt string` (all with camelCase json tags)
   - And exposes: `GetContentByPlacements(ctx context.Context, placements []string) (map[string][]ContentResult, error)`, `FlushCache(ctx context.Context, placements []string) error`

3. **AC-3: RuleEvaluator interface and Registry defined in evaluator package**
   - Given a new file `internal/service/evaluator/evaluator.go`
   - When it is compiled
   - Then `RuleEvaluator` interface exposes: `Evaluate(ctx context.Context, rule entity.DecisionRule) (float64, error)`, `RuleType() string`
   - And `Registry` struct with `NewRegistry()`, `Register(RuleEvaluator)`, `Get(ruleType string) (RuleEvaluator, bool)` is implemented

4. **AC-4: Three concrete RuleEvaluator implementations exist**
   - Given files in `internal/service/evaluator/`
   - When compiled
   - Then `ScoringEvaluator` returns `rule.Score` directly (type: `"SCORING"`)
   - And `SegmentEvaluator` returns `rule.Score` with a TODO comment for segment matching (type: `"SEGMENT"`)
   - And `EligibleEvaluator` returns `rule.Score` with a TODO comment for eligibility check (type: `"ELIGIBLE"`)
   - And all three satisfy the `RuleEvaluator` interface

5. **AC-5: `go build ./...` passes with zero errors**
   - Given all new files are added
   - When `go build ./...` is run
   - Then there are no compilation errors

6. **AC-6: All new files follow project conventions**
   - Given the existing codebase patterns
   - When new files are reviewed
   - Then package names, file naming, comment style, and interface placement all follow the project's established patterns

## Tasks / Subtasks

- [x] **Task 1: Create `internal/domain/service/` package with RuntimeService interface** (AC: #1, #5, #6)
  - [x] 1.1 Create directory `internal/domain/service/` (package `service`)
  - [x] 1.2 Create `internal/domain/service/cms_runtime.go` — `RuntimeService` interface with `EvaluateAll`, `EvaluatePlacement`, `Start`, `Stop`

- [x] **Task 2: Create DeliveryService interface and ContentResult struct** (AC: #2, #5, #6)
  - [x] 2.1 Create `internal/domain/service/cms_delivery.go` — `ContentResult` struct + `DeliveryService` interface

- [x] **Task 3: Create evaluator package with RuleEvaluator interface and Registry** (AC: #3, #5, #6)
  - [x] 3.1 Create directory `internal/service/evaluator/` (package `evaluator`)
  - [x] 3.2 Create `internal/service/evaluator/evaluator.go` — `RuleEvaluator` interface + `Registry` struct

- [x] **Task 4: Implement three concrete evaluators** (AC: #4, #5, #6)
  - [x] 4.1 Create `internal/service/evaluator/scoring_evaluator.go` — `ScoringEvaluator`
  - [x] 4.2 Create `internal/service/evaluator/segment_evaluator.go` — `SegmentEvaluator`
  - [x] 4.3 Create `internal/service/evaluator/eligible_evaluator.go` — `EligibleEvaluator`

- [x] **Task 5: Verify compilation** (AC: #5)
  - [x] 5.1 Run `go build ./...` — confirm zero errors

## Dev Notes

### Source Reference

- **Design Thinking Session:** `_bmad-output/design-thinking-2026-04-08.md`
  - Prototype section: Domain Service Interfaces (#1), Rule Evaluator Strategy (#2)
  - Top Concepts: Concept A (Strategy Pattern), Concept F (Shared Domain, Separate Services)

### Architecture Context

This story establishes the interface contracts only — no implementation logic yet. The full cms-runtime service implementation follows in Story 2.2+.

**Service boundary approach (Concept F — shared domain, separate service layers):**
```
internal/
├── domain/
│   └── service/        ← NEW in this story: RuntimeService, DeliveryService interfaces
├── service/
│   ├── evaluator/      ← NEW in this story: RuleEvaluator interface + Registry + 3 evaluators
```

**Why domain/service/ and NOT domain/repository/?**
- `domain/repository/` holds data-access contracts (ScheduleRepository, CacheRepository)
- `domain/service/` holds business-logic contracts (RuntimeService, DeliveryService)
- This separation is the project's clean architecture boundary

### Project Conventions (MUST FOLLOW)

**Interface pattern** — follow existing repo interfaces style (see `internal/domain/repository/cache.go`):
```go
package service

import "context"

// RuntimeService defines the contract for ...
type RuntimeService interface {
    // MethodName verb phrase.
    MethodName(ctx context.Context, ...) error
}
```

**Struct pattern** — follow existing entity style (camelCase json tags, no gorm tags needed for DTOs):
```go
type ContentResult struct {
    ContentPath string  `json:"contentPath"`
    AemURL      string  `json:"aemUrl"`
    Score       float64 `json:"score"`
    RuleID      string  `json:"ruleId"`
    RuleType    string  `json:"ruleType"`
    EvaluatedAt string  `json:"evaluatedAt"`
}
```

**Evaluator concrete struct pattern:**
```go
// ScoringEvaluator evaluates a DecisionRule of type SCORING.
// It returns the rule's pre-assigned score directly.
type ScoringEvaluator struct{}

func (e *ScoringEvaluator) Evaluate(ctx context.Context, rule entity.DecisionRule) (float64, error) {
    return rule.Score, nil
}

func (e *ScoringEvaluator) RuleType() string { return string(enums.DecisionRuleTypeScoring) }
```

Use `enums.DecisionRuleTypeScoring` (= `"SCORING"`), `enums.DecisionRuleTypeSegment`, `enums.DecisionRuleTypeEligible` for the `RuleType()` return values to stay consistent with the enum definitions in `internal/domain/entity/enums/decision_rule_type.go`.

**File naming** — snake_case: `cms_runtime.go`, `cms_delivery.go`, `evaluator.go`, `scoring_evaluator.go`

**Package imports** — evaluator package needs:
```go
import (
    "context"
    "kbank-ecms/internal/domain/entity"
    "kbank-ecms/internal/domain/entity/enums"
)
```

### Existing Code to Reference

- `internal/domain/repository/cache.go` — CacheRepository interface (model for interface style)
- `internal/domain/repository/schedule.go` — ScheduleRepository interface (model for interface style)
- `internal/domain/entity/decision_rule.go` — `DecisionRule` struct with `Score float64`, `Type enums.DecisionRuleType`, `ContentPath string`
- `internal/domain/entity/enums/decision_rule_type.go` — `DecisionRuleTypeScoring = "SCORING"`, `DecisionRuleTypeSegment = "SEGMENT"`, `DecisionRuleTypeEligible = "ELIGIBLE"`
- `internal/service/schedule_service.go` — example of existing service in `internal/service/`

### Key Design Decisions from Testing Phase

- **cms-delivery is a pure Redis reader** — `DeliveryService` does NOT declare any PostgreSQL dependency; no `loader` fallback in the interface
- **cms-runtime is write-only to Redis** — `RuntimeService` writes; `DeliveryService` reads; never reversed
- **RuleType() returns enum string value** — use `string(enums.DecisionRuleTypeScoring)` not hardcoded `"SCORING"` literals in struct files
- **Evaluators hold no state** — all three concrete evaluators are stateless structs (`struct{}`)

### Module Name

The Go module is `kbank-ecms` (see `go.mod`). All imports use this prefix.

---

## Dev Agent Record

### Implementation Plan

_To be filled during development_

### Debug Log

_To be filled during development_

### Completion Notes

All 6 files created successfully. `go build ./...` passes with zero errors.

- `RuntimeService` interface: 4 methods covering batch evaluation, placement-level evaluation, and ticker lifecycle.
- `DeliveryService` interface: pure Redis reader — no PostgreSQL dependency as validated in the Test phase (Assumption A4 invalidated).
- `ContentResult` struct: 6 fields with camelCase json tags matching the design doc.
- `RuleEvaluator` + `Registry`: strategy pattern with map-keyed dispatch; all three evaluators are stateless `struct{}` values using `string(enums.DecisionRuleTypeX)` for type identity.
- Note: concrete evaluator methods use value receivers (not pointer receivers) since they are stateless.

---

## File List

- `internal/domain/service/cms_runtime.go` — `RuntimeService` interface (package `service`)
- `internal/domain/service/cms_delivery.go` — `ContentResult` struct + `DeliveryService` interface (package `service`)
- `internal/service/evaluator/evaluator.go` — `RuleEvaluator` interface + `Registry` struct (package `evaluator`)
- `internal/service/evaluator/scoring_evaluator.go` — `ScoringEvaluator` (package `evaluator`)
- `internal/service/evaluator/segment_evaluator.go` — `SegmentEvaluator` (package `evaluator`)
- `internal/service/evaluator/eligible_evaluator.go` — `EligibleEvaluator` (package `evaluator`)

---

### Review Findings

- [x] [Review][Patch] P1: "read-only" comment contradicts FlushCache method [internal/domain/service/cms_delivery.go:15-16]
- [x] [Review][Patch] P2: GetContentByPlacements nil vs empty slice return behavior unspecified in comment [internal/domain/service/cms_delivery.go:20-23]
- [x] [Review][Patch] P3: FlushCache nil vs empty slice behavior ambiguous — "all placement caches flushed" vs per-placement [internal/domain/service/cms_delivery.go:25-27]
- [x] [Review][Patch] P4: ContentResult.EvaluatedAt field has no time format specification (RFC3339? Unix?) [internal/domain/service/cms_delivery.go:12]
- [x] [Review][Patch] P5: EvaluateAll comment uses undefined terms "active placements" and "current time window" [internal/domain/service/cms_runtime.go:9-10]
- [x] [Review][Patch] P6: Start() context parameter relationship to ticker goroutine not documented [internal/domain/service/cms_runtime.go:15-16]
- [x] [Review][Patch] P7: Stop() graceful shutdown semantics not documented (blocks? timeout? idempotent?) [internal/domain/service/cms_runtime.go:18-19]
- [x] [Review][Defer] D1: nil guard missing on Registry.Register(e RuleEvaluator) — panic if e is nil [internal/service/evaluator/evaluator.go:32] — deferred, belongs in implementation wiring (Story 2.2+)
- [x] [Review][Defer] D2: empty string guard missing on Registry.Get(ruleType string) [internal/service/evaluator/evaluator.go:37] — deferred, belongs in implementation (Story 2.2+)
- [x] [Review][Defer] D3: NaN/Inf score propagation from all three Evaluate() placeholders — no guard [evaluator/*.go] — deferred, DB constraints prevent this in practice; validate at cache-write layer in Story 2.2
- [x] [Review][Defer] D4: context cancellation not pre-checked in EvaluateAll/EvaluatePlacement — deferred, implementation concern (Story 2.2+)
- [x] [Review][Defer] D5: No input size limit on GetContentByPlacements placementNames slice — deferred, HTTP handler/middleware concern (Story 2.3)
- [x] [Review][Defer] D6: RuntimeService Start/Stop idempotency not guaranteed by contract — deferred, implementation lifecycle management (Story 2.2+)

---

## Change Log

| Date | Change |
|------|--------|
| 2026-04-08 | Story created from Design Thinking session design-thinking-2026-04-08.md |
| 2026-04-08 | Story implemented — all 6 files created, `go build ./...` passes, status → review |
| 2026-04-08 | Code review complete — 7 patch fixed, 6 defer, 8 dismissed; status → done |
