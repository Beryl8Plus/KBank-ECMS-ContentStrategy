# Design Thinking Session: CMS Runtime & Delivery Service Design

**Date:** 2026-04-08
**Facilitator:** Nrtdemo
**Design Challenge:** Designing two new logical services (cms-runtime, cms-delivery-service) within the KBank-ECMS-Backend monolith to enable real-time decision rule evaluation and cached content delivery for an orchestrator microservice.

---

## 🎯 Design Challenge

The KBank-ECMS-Backend monolith currently has a placeholder `RuleManagementService` and no dedicated content delivery mechanism. An orchestrator microservice needs to:

1. **Evaluate decision rules** (content personalization, promotion/campaign targeting, eligibility/business rules) at runtime and cache evaluated results in Redis for fast repeated access.
2. **Serve Adobe Experience Manager (AEM) content paths** per placement from Redis cache, with automatic fallback to PostgreSQL when cache is empty — then populate the cache for subsequent requests.

### Constraints & Context

| Dimension           | Detail                                                                                                                                              |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Architecture**    | Monolith (Go, Gin, GORM, Redis, PostgreSQL) — services are logical boundaries, not separate deployments                                             |
| **Consumer**        | Orchestrator microservice (not end-users directly)                                                                                                  |
| **Rule Types**      | Content personalization, Promotion/campaign targeting, Eligibility/business rules                                                                   |
| **Content Source**  | Adobe Experience Manager (AEM) — content paths per placement                                                                                        |
| **Latency Target**  | Standard (<200ms)                                                                                                                                   |
| **Existing Infra**  | Redis (cache with `GetSet` cache-through pattern), PostgreSQL via GORM, Azure integration                                                           |
| **Existing Assets** | `DecisionRule` entity, `Schedule` entity (rule→placement binding with recurrence), `CacheRepository` interface, placeholder `RuleManagementService` |

### Challenge Statement

> **How might we design two cohesive internal services — cms-runtime and cms-delivery-service — within the existing Go monolith, so that the orchestrator microservice can efficiently evaluate decision rules and retrieve AEM content paths with <200ms latency, high cache hit rates, decoupled rule evaluation logic, and a clear path for future service extraction?**

### Data Flow

| Service                  | Responsibility                                                                             | Data Flow                                                                         |
| ------------------------ | ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------- |
| **cms-runtime**          | Evaluate decision rules (personalization, targeting, eligibility) → cache results in Redis | cms-runtime → cms-delivery-service → Rule Engine → Redis (write)                  |
| **cms-delivery-service** | Serve AEM content paths per placement from Redis, fallback to PostgreSQL                   | Orchestrator → cms-delivery → Redis (read) → PostgreSQL (miss) → Redis (populate) |

---

## 👥 EMPATHIZE: Understanding Users

### User Insights

**Three user archetypes emerged:**

1. **The Orchestrator Microservice** (machine consumer)
   - Currently calls external AEM API directly for content — no rule evaluation, no caching
   - Sends placement names like `wsaHomeBanner`, `wsaPortBanner`, `wsaSplash`, `wsaLandingPage`
   - Needs content paths (AEM URLs) returned per placement, ranked by rule score (Top N)
   - Expects user context/attributes as input for personalization

2. **The Development Team** (builders & maintainers)
   - `RuleManagementService` exists as a placeholder — no rule evaluation engine yet
   - Redis infrastructure exists with `GetSet` cache-through pattern, but no content caching layer is built
   - Need clean service boundaries within the monolith for future extraction
   - Need clear API contracts between cms-runtime, cms-delivery, and the orchestrator

3. **The Business Stakeholders** (campaign managers, content owners)
   - Need content personalization, promotion targeting, and eligibility rules to work together
   - Schedules already bind rules to placements — this needs to drive the evaluation
   - Cache invalidation should trigger when schedules are created/updated, plus TTL and manual flush

### Key Observations

| Observation                                                   | Implication                                                                                                       |
| ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **No rule evaluation engine exists**                          | cms-runtime is greenfield — must design rule matching, scoring, and ranking from scratch                          |
| **No caching layer for content**                              | cms-delivery-service must implement cache-aside/through pattern for AEM content paths                             |
| **Orchestrator currently calls AEM directly**                 | cms-delivery replaces direct AEM calls, adding Redis caching in front                                             |
| **Evaluation is batch pre-compute**                           | cms-runtime runs on a schedule (not per-request), pre-computes winning content paths, and writes results to Redis |
| **Top N results per placement**                               | Rule engine must sort by score and return ranked list, not just a single winner                                   |
| **Placements are named slots**                                | `wsaHomeBanner`, `wsaPortBanner`, `wsaSplash`, `wsaLandingPage` — Redis keys can be structured around these       |
| **Cache invalidation: TTL + schedule changes + manual flush** | Three invalidation strategies must coexist                                                                        |

### Empathy Map Summary

```
┌─────────────────────────────────────────────────────────────┐
│                     ORCHESTRATOR SERVICE                     │
├──────────────────────┬──────────────────────────────────────┤
│       SAYS           │              THINKS                   │
│ "Give me content     │ "I need fast, reliable content paths" │
│  paths for these     │ "I shouldn't have to call AEM every"  │
│  placements with     │  time — that's slow and fragile"      │
│  user context"       │ "Rules should be pre-evaluated"       │
├──────────────────────┼──────────────────────────────────────┤
│       DOES           │              FEELS                    │
│ Calls AEM API        │ Frustrated: no caching, no rules     │
│ directly today       │ Blocked: waiting for rule engine      │
│ Passes placement     │ Uncertain: no API contract defined    │
│ names as input       │ Constrained: latency depends on AEM  │
└──────────────────────┴──────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                     DEVELOPMENT TEAM                         │
├──────────────────────┬──────────────────────────────────────┤
│       SAYS           │              THINKS                   │
│ "We have the entity  │ "The monolith needs clear boundaries" │
│  models and Redis    │ "We already have cache-through in     │
│  infra ready"        │  Redis — we can build on that"        │
├──────────────────────┼──────────────────────────────────────┤
│       DOES           │              FEELS                    │
│ Built Schedule +     │ Ready: infrastructure exists          │
│ DecisionRule models  │ Cautious: need clean separation for   │
│ Placeholder service  │  future microservice extraction       │
│ Redis GetSet ready   │ Motivated: clear problem to solve     │
└──────────────────────┴──────────────────────────────────────┘
```

---

## 🎨 DEFINE: Frame the Problem

### Point of View Statements

**POV 1 — The Orchestrator (Machine Consumer)**

> The orchestrator microservice needs pre-computed, ranked content paths per placement available in Redis because calling AEM directly on every request is slow, fragile, and prevents rule-based personalization.

**POV 2 — The Rule Engine (cms-runtime)**

> The cms-runtime service needs to batch-evaluate decision rules against schedules and placements because no rule evaluation engine exists, and the business needs personalization, targeting, and eligibility decisions applied automatically via an internal Go ticker/cron.

**POV 3 — The Development Team**

> The development team needs cleanly separated internal service boundaries with clear API contracts because the monolith must support future extraction into independent microservices without rewriting.

### How Might We Questions

**cms-runtime (Batch Pre-compute — Write-only to Redis)**

1. **HMW** pre-compute rule evaluations efficiently so Redis always has fresh, ranked content paths ready?
2. **HMW** handle multiple rule types (personalization, targeting, eligibility) in a single evaluation pass?
3. **HMW** tie evaluation triggers to schedule lifecycle (create/update) alongside the internal Go ticker?
4. **HMW** ensure the rule engine scores and ranks Top N results per placement deterministically?

**cms-delivery-service (Read from Redis — PostgreSQL Fallback)**

5. **HMW** serve content paths with <200ms latency while supporting cache miss → PostgreSQL → cache populate?
6. **HMW** structure Redis keys around named placements (`wsaHomeBanner`, etc.) so lookups are O(1)?
7. **HMW** support three coexisting invalidation strategies (TTL, schedule-triggered, manual flush)?
8. **HMW** make the delivery API contract simple for the orchestrator while returning Top N ranked results?

**Cross-cutting**

9. **HMW** keep service boundaries clean within the monolith for future microservice extraction?
10. **HMW** make the data flow (cms-runtime → Redis ← cms-delivery) observable and debuggable?

### Key Insights

| Insight                                      | Design Decision                                                                          |
| -------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **cms-runtime is write-only to Redis**       | Clear separation: runtime writes, delivery reads — no bidirectional coupling             |
| **Batch pre-compute via internal Go ticker** | cms-runtime runs as a background goroutine with configurable interval, not external cron |
| **All 10 HMW questions are priority**        | The full pipeline must be designed holistically — no area can be deferred                |
| **Placements are named slots**               | Redis key design: `placement:{name}` → sorted set or list of ranked content paths        |
| **Top N results per placement**              | Redis sorted sets (`ZREVRANGE`) are a natural fit for score-ranked content paths         |
| **Three invalidation strategies coexist**    | TTL on keys + publish invalidation on schedule events + manual flush endpoint            |
| **Future extraction readiness**              | Service interfaces must be Go interfaces with no shared mutable state                    |

### Validated Architecture Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                        cms-runtime (Background)                   │
│                                                                    │
│  Internal Go Ticker ──→ Load active schedules (PostgreSQL)        │
│          │                                                         │
│          ├──→ For each placement:                                  │
│          │      1. Find matching DecisionRules                     │
│          │      2. Evaluate rules (personalization/targeting/elig) │
│          │      3. Score & rank → Top N                            │
│          │      4. Write ranked content paths → Redis              │
│          │         Key: placement:{name}                           │
│          │         Value: sorted set (score → content_path/AEM URL)│
│          │         TTL: configurable                               │
│          │                                                         │
│  Schedule lifecycle (create/update) ──→ Trigger re-evaluation     │
└──────────────────────────────────────────────────────────────────┘
                              │
                         Redis (bridge)
                              │
┌──────────────────────────────────────────────────────────────────┐
│                     cms-delivery-service (HTTP)                    │
│                                                                    │
│  Orchestrator ──→ GET /delivery?placements=wsaHomeBanner,...       │
│          │                                                         │
│          ├──→ Redis GET placement:{name} (O(1) lookup)            │
│          │      ├─ HIT  → return Top N content paths              │
│          │      └─ MISS → PostgreSQL query → populate Redis → ret │
│          │                                                         │
│  Manual flush ──→ POST /delivery/cache/flush                      │
└──────────────────────────────────────────────────────────────────┘
```

---

## 💡 IDEATE: Generate Solutions

### Selected Methods

- **SCAMPER Design** — applied lenses to existing monolith patterns (Redis GetSet, GORM, Strategy)
- **Analogous Inspiration** — borrowed from CQRS (write/read split), cache-aside, strategy pattern
- **Brainstorming** — 30 ideas generated across 6 design areas, clustered and scored

### Generated Ideas

30 ideas across 6 areas: Rule Engine Architecture (5), Ticker & Scheduling (5), Redis Key Design (5), Cache + Fallback Pattern (5), API Contract (5), Service Boundary (5).

### Top Concepts

**Concept A: Rule Evaluation — Strategy Pattern per Rule Type (#2)**

Each rule type (personalization, targeting, eligibility) implements a `RuleEvaluator` interface. A registry dispatches rules to the correct evaluator by type. New rule types are added by implementing the interface and registering.

```go
type RuleEvaluator interface {
    Evaluate(ctx context.Context, rule DecisionRule, context UserContext) (score float64, err error)
    RuleType() string
}
```

Evaluators registered at startup:

- `PersonalizationEvaluator` — matches user attributes to rule conditions
- `TargetingEvaluator` — campaign/promotion targeting logic
- `EligibilityEvaluator` — pass/fail eligibility checks (score = max if pass, 0 if fail)

**Concept B: Ticker — Reactive + Periodic Hybrid (#8)**

- **Periodic**: Background goroutine with `time.Ticker` (configurable interval, e.g., 5 min) re-evaluates all active placements
- **Reactive**: Hook into `ScheduleService.CreateSchedule()` and `UpdateSchedule()` to trigger immediate re-evaluation for affected placements
- Benefits: Fresh cache after schedule changes + guaranteed periodic refresh

```
┌─ time.Ticker (every N min) ──→ EvaluateAll()
│
├─ ScheduleService.Create() ──→ EvaluatePlacement(placement_id)
│
└─ ScheduleService.Update() ──→ EvaluatePlacement(placement_id)
```

**Concept C: Redis Structure — Content Path as Key, JSON Data Layer**

Redis key is the `content_path`, and the value is a JSON object with layered metadata:

```
Key:   content_path:{aem_content_path}
Value: {
    "content_path": "/content/kbank/homepage/banner-spring-2026",
    "placement": "wsaHomeBanner",
    "rule_id": "uuid",
    "rule_type": "personalization",
    "score": 85.5,
    "rank": 1,
    "aem_url": "https://aem.kbank.co.th/content/...",
    "metadata": { "locale": "th", "variant": "A" },
    "evaluated_at": "2026-04-08T10:00:00Z"
}
```

Additionally, a placement index key maps placements to their content paths:

```
Key:   placement:{name}  (e.g., placement:wsaHomeBanner)
Value: ["content_path:/content/kbank/.../banner1", "content_path:/content/kbank/.../banner2", ...]
       (ordered by score, Top N)
```

Lookup flow:

1. `GET placement:{name}` → list of content_path keys
2. `MGET content_path:{path1}, content_path:{path2}, ...` → JSON data per path

**Concept D: Cache Pattern — Existing GetSet Cache-Through (#17)**

Leverage the existing `CacheRepository.GetSet(key, ttl, loader)` pattern:

- `key` = `placement:{name}` or `content_path:{path}`
- `ttl` = configurable per placement or global
- `loader` = function that queries PostgreSQL for the content path data

```go
result, err := cacheRepo.GetSet(placementKey, ttl, func() (interface{}, error) {
    return postgresRepo.FindContentPathsByPlacement(ctx, placement)
})
```

**Concept E: API Contract — GET with Query Params (#21)**

```
GET /delivery?placements=wsaHomeBanner,wsaPortBanner,wsaSplash

Response 200:
{
    "wsaHomeBanner": [
        {"content_path": "/content/kbank/.../banner1", "score": 95.0, "aem_url": "https://..."},
        {"content_path": "/content/kbank/.../banner2", "score": 85.5, "aem_url": "https://..."}
    ],
    "wsaPortBanner": [
        {"content_path": "/content/kbank/.../port1", "score": 90.0, "aem_url": "https://..."}
    ],
    "wsaSplash": []  // no active rules
}
```

Cache flush:

```
POST /delivery/cache/flush
POST /delivery/cache/flush?placements=wsaHomeBanner  // selective flush
```

**Concept F: Service Boundary — Shared Domain, Separate Service Layers (#28)**

```
internal/
├── domain/
│   └── entity/          ← shared: DecisionRule, Schedule, Placement, etc.
│   └── repository/      ← shared interfaces: CacheRepository, DatabaseRepository
│   └── service/         ← NEW: RuntimeService, DeliveryService interfaces
│
├── service/
│   ├── cms_runtime_service.go        ← implements RuntimeService
│   └── cms_delivery_service.go       ← implements DeliveryService
│
├── delivery/
│   └── http/
│       └── handler/
│           ├── cms_delivery_handler.go   ← GET /delivery
│           └── cms_cache_handler.go      ← POST /delivery/cache/flush
│
├── repository/
│   └── ... (existing + new content-path queries)
```

---

## 🛠️ PROTOTYPE: Make Ideas Tangible

### Prototype Approach

**Storyboarding** (full lifecycle visualization) + **Paper Prototyping** (Go interfaces, Redis key layouts, API contract). The prototype is code-level design, grounded in the existing entity models (`DecisionRule`, `Schedule`, `Placement`) and existing interfaces (`CacheRepository`, `ScheduleRepository`).

### Prototype Description

#### 1. Domain Service Interfaces (new: `internal/domain/service/`)

```go
// internal/domain/service/cms_runtime.go
package service

import "context"

// RuntimeService defines the cms-runtime contract.
// Responsible for batch evaluation of decision rules and writing results to Redis.
type RuntimeService interface {
    // EvaluateAll runs a full evaluation pass across all active placements.
    // Called by the internal ticker on each interval.
    EvaluateAll(ctx context.Context) error

    // EvaluatePlacement re-evaluates a single placement.
    // Called reactively when a schedule is created or updated.
    EvaluatePlacement(ctx context.Context, placementName string) error

    // Start begins the background ticker goroutine.
    Start(ctx context.Context) error

    // Stop gracefully shuts down the ticker.
    Stop() error
}
```

```go
// internal/domain/service/cms_delivery.go
package service

import "context"

// ContentResult represents a single ranked content path for a placement.
type ContentResult struct {
    ContentPath string  `json:"contentPath"`
    AemURL      string  `json:"aemUrl"`
    Score       float64 `json:"score"`
    RuleID      string  `json:"ruleId"`
    RuleType    string  `json:"ruleType"`
    EvaluatedAt string  `json:"evaluatedAt"`
}

// DeliveryService defines the cms-delivery-service contract.
// Responsible for serving content paths from Redis with PostgreSQL fallback.
type DeliveryService interface {
    // GetContentByPlacements returns Top N ranked content paths for each placement.
    // Uses Redis cache with PostgreSQL fallback via GetSet pattern.
    GetContentByPlacements(ctx context.Context, placements []string) (map[string][]ContentResult, error)

    // FlushCache invalidates cache for specified placements, or all if empty.
    FlushCache(ctx context.Context, placements []string) error
}
```

#### 2. Rule Evaluator — Strategy Pattern (new: `internal/service/evaluator/`)

```go
// internal/service/evaluator/evaluator.go
package evaluator

import (
    "context"
    "kbank-ecms/internal/domain/entity"
)

// UserContext holds the attributes sent by the orchestrator for rule evaluation.
type UserContext struct {
    UserID     string            `json:"userId"`
    Segment    string            `json:"segment"`
    Attributes map[string]string `json:"attributes"`
}

// RuleEvaluator evaluates a single decision rule against user context.
type RuleEvaluator interface {
    // Evaluate returns a score for the rule. Returns 0 if rule does not match.
    Evaluate(ctx context.Context, rule entity.DecisionRule) (float64, error)

    // RuleType returns the DecisionRuleType this evaluator handles.
    RuleType() string
}

// Registry maps rule types to their evaluators.
type Registry struct {
    evaluators map[string]RuleEvaluator
}

func NewRegistry() *Registry {
    return &Registry{evaluators: make(map[string]RuleEvaluator)}
}

func (r *Registry) Register(e RuleEvaluator) {
    r.evaluators[e.RuleType()] = e
}

func (r *Registry) Get(ruleType string) (RuleEvaluator, bool) {
    e, ok := r.evaluators[ruleType]
    return e, ok
}
```

Concrete evaluators:

```go
// SCORING evaluator — uses the rule's score directly
type ScoringEvaluator struct{}
func (e *ScoringEvaluator) Evaluate(ctx context.Context, rule entity.DecisionRule) (float64, error) {
    return rule.Score, nil
}
func (e *ScoringEvaluator) RuleType() string { return "SCORING" }

// SEGMENT evaluator — matches user segment, returns score if match
type SegmentEvaluator struct{}
func (e *SegmentEvaluator) Evaluate(ctx context.Context, rule entity.DecisionRule) (float64, error) {
    // TODO: match rule conditions against user segment from context
    return rule.Score, nil
}
func (e *SegmentEvaluator) RuleType() string { return "SEGMENT" }

// ELIGIBLE evaluator — pass/fail check, max score if eligible, 0 otherwise
type EligibleEvaluator struct{}
func (e *EligibleEvaluator) Evaluate(ctx context.Context, rule entity.DecisionRule) (float64, error) {
    // TODO: check eligibility conditions
    return rule.Score, nil // or 0 if not eligible
}
func (e *EligibleEvaluator) RuleType() string { return "ELIGIBLE" }
```

#### 3. cms-runtime Service Implementation

```go
// internal/service/cms_runtime_service.go
package service

import (
    "context"
    "encoding/json"
    "sort"
    "time"

    domainRepo "kbank-ecms/internal/domain/repository"
    domainSvc  "kbank-ecms/internal/domain/service"
    "kbank-ecms/internal/service/evaluator"
)

type cmsRuntimeService struct {
    scheduleRepo domainRepo.ScheduleRepository
    cacheRepo    domainrepo.RedisCacheRepository
    dbRepo       domainRepo.DatabaseRepository
    registry     *evaluator.Registry
    ticker       *time.Ticker
    interval     time.Duration
    cacheTTL     time.Duration
    topN         int
    stopCh       chan struct{}
}

func NewCMSRuntimeService(
    scheduleRepo domainRepo.ScheduleRepository,
    cacheRepo domainrepo.RedisCacheRepository,
    dbRepo domainRepo.DatabaseRepository,
    registry *evaluator.Registry,
    interval, cacheTTL time.Duration,
    topN int,
) domainSvc.RuntimeService {
    return &cmsRuntimeService{
        scheduleRepo: scheduleRepo,
        cacheRepo:    cacheRepo,
        dbRepo:       dbRepo,
        registry:     registry,
        interval:     interval,
        cacheTTL:     cacheTTL,
        topN:         topN,
        stopCh:       make(chan struct{}),
    }
}

func (s *cmsRuntimeService) Start(ctx context.Context) error {
    s.ticker = time.NewTicker(s.interval)
    go func() {
        for {
            select {
            case <-s.ticker.C:
                _ = s.EvaluateAll(ctx)
            case <-s.stopCh:
                return
            }
        }
    }()
    return nil
}

func (s *cmsRuntimeService) Stop() error {
    s.ticker.Stop()
    close(s.stopCh)
    return nil
}

func (s *cmsRuntimeService) EvaluateAll(ctx context.Context) error {
    // 1. Load all active schedules with DecisionRule + Placement preloaded
    schedules, err := s.scheduleRepo.ListSchedules(ctx)
    if err != nil {
        return err
    }

    // 2. Group schedules by placement name
    placementSchedules := make(map[string][]*entity.Schedule)
    for _, sch := range schedules {
        if sch.IsActive && sch.Placement != nil {
            placementSchedules[sch.Placement.Name] = append(
                placementSchedules[sch.Placement.Name], sch,
            )
        }
    }

    // 3. For each placement, evaluate and rank
    for placementName, scheds := range placementSchedules {
        if err := s.evaluateAndCache(ctx, placementName, scheds); err != nil {
            // log error, continue with next placement
            continue
        }
    }
    return nil
}

func (s *cmsRuntimeService) EvaluatePlacement(ctx context.Context, placementName string) error {
    // Load schedules for this specific placement and re-evaluate
    // (implementation queries schedules filtered by placement name)
    return nil
}

func (s *cmsRuntimeService) evaluateAndCache(
    ctx context.Context,
    placementName string,
    schedules []*entity.Schedule,
) error {
    type scored struct {
        Result domainSvc.ContentResult
        Score  float64
    }

    var results []scored

    for _, sch := range schedules {
        if sch.DecisionRule == nil {
            continue
        }
        rule := sch.DecisionRule

        eval, ok := s.registry.Get(string(rule.Type))
        if !ok {
            continue
        }

        score, err := eval.Evaluate(ctx, *rule)
        if err != nil || score <= 0 {
            continue
        }

        results = append(results, scored{
            Score: score,
            Result: domainSvc.ContentResult{
                ContentPath: rule.ContentPath,
                Score:       score,
                RuleID:      rule.ID.String(),
                RuleType:    string(rule.Type),
                EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
            },
        })
    }

    // Sort by score descending, take Top N
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    if len(results) > s.topN {
        results = results[:s.topN]
    }

    // Build final content results
    contentResults := make([]domainSvc.ContentResult, len(results))
    for i, r := range results {
        contentResults[i] = r.Result
    }

    // Write to Redis as JSON: key = "placement:{name}"
    data, _ := json.Marshal(contentResults)
    return s.cacheRepo.Set(ctx, "placement:"+placementName, string(data), s.cacheTTL)
}
```

#### 4. cms-delivery Service Implementation

```go
// internal/service/cms_delivery_service.go
package service

import (
    "context"
    "encoding/json"

    domainRepo "kbank-ecms/internal/domain/repository"
    domainSvc  "kbank-ecms/internal/domain/service"
)

type cmsDeliveryService struct {
    cacheRepo domainrepo.RedisCacheRepository
    cacheTTL  time.Duration
}

func NewCMSDeliveryService(
    cacheRepo domainrepo.RedisCacheRepository,
    cacheTTL time.Duration,
) domainSvc.DeliveryService {
    return &cmsDeliveryService{
        cacheRepo: cacheRepo,
        cacheTTL:  cacheTTL,
    }
}

func (s *cmsDeliveryService) GetContentByPlacements(
    ctx context.Context,
    placements []string,
) (map[string][]domainSvc.ContentResult, error) {
    result := make(map[string][]domainSvc.ContentResult, len(placements))

    for _, placement := range placements {
        key := "placement:" + placement

        // Use existing GetSet cache-through pattern
        // On cache hit: returns cached JSON
        // On cache miss: loader queries PostgreSQL, populates cache, returns
        data, err := s.cacheRepo.GetSet(ctx, key, s.cacheTTL, func(ctx context.Context) (string, error) {
            return s.loadFromPostgreSQL(ctx, placement)
        })
        if err != nil {
            result[placement] = []domainSvc.ContentResult{}
            continue
        }

        var items []domainSvc.ContentResult
        if err := json.Unmarshal([]byte(data), &items); err != nil {
            result[placement] = []domainSvc.ContentResult{}
            continue
        }
        result[placement] = items
    }
    return result, nil
}

func (s *cmsDeliveryService) loadFromPostgreSQL(
    ctx context.Context,
    placement string,
) (string, error) {
    // Query PostgreSQL: join schedules + decision_rules + placements
    // WHERE placement.name = ? AND schedule.is_active = true
    //   AND schedule.effective_from <= now AND schedule.effective_until > now
    // ORDER BY decision_rule.score DESC LIMIT topN
    // Marshal result as JSON string
    return "[]", nil // placeholder
}

func (s *cmsDeliveryService) FlushCache(ctx context.Context, placements []string) error {
    if len(placements) == 0 {
        return s.cacheRepo.FlushDB(ctx)
    }
    // Selective flush: delete each placement key
    for _, p := range placements {
        // Use Set with 0 TTL or implement Delete
        _ = s.cacheRepo.Set(ctx, "placement:"+p, "", 1) // expire immediately
    }
    return nil
}
```

#### 5. HTTP Handler — Delivery Endpoint

```go
// internal/delivery/http/handler/cms_delivery_handler.go
package handler

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    domainSvc "kbank-ecms/internal/domain/service"
)

type CMSDeliveryHandler struct {
    deliverySvc domainSvc.DeliveryService
}

func NewCMSDeliveryHandler(svc domainSvc.DeliveryService) *CMSDeliveryHandler {
    return &CMSDeliveryHandler{deliverySvc: svc}
}

// GetContent handles GET /delivery?placements=wsaHomeBanner,wsaPortBanner
func (h *CMSDeliveryHandler) GetContent(c *gin.Context) {
    placementsParam := c.Query("placements")
    if placementsParam == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "placements query parameter required"})
        return
    }

    placements := strings.Split(placementsParam, ",")
    result, err := h.deliverySvc.GetContentByPlacements(c.Request.Context(), placements)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, result)
}

// FlushCache handles POST /delivery/cache/flush?placements=wsaHomeBanner
func (h *CMSDeliveryHandler) FlushCache(c *gin.Context) {
    placementsParam := c.Query("placements")
    var placements []string
    if placementsParam != "" {
        placements = strings.Split(placementsParam, ",")
    }

    if err := h.deliverySvc.FlushCache(c.Request.Context(), placements); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "flushed"})
}
```

#### 6. Redis Key Layout

```
┌──────────────────────────────────────────────────────────────────┐
│                         Redis Key Design                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  Key Pattern:     placement:{name}                                │
│  Example:         placement:wsaHomeBanner                         │
│  Type:            String (JSON)                                    │
│  TTL:             Configurable (e.g., 300s / 5 min)               │
│                                                                    │
│  Value (JSON array, ordered by score descending):                 │
│  [                                                                 │
│    {                                                               │
│      "contentPath": "/content/kbank/homepage/banner-spring-2026", │
│      "aemUrl": "",                                                │
│      "score": 95.5,                                               │
│      "ruleId": "a1b2c3d4-...",                                    │
│      "ruleType": "SCORING",                                      │
│      "evaluatedAt": "2026-04-08T10:00:00Z"                       │
│    },                                                              │
│    {                                                               │
│      "contentPath": "/content/kbank/homepage/banner-promo-q2",    │
│      "aemUrl": "",                                                │
│      "score": 85.0,                                               │
│      "ruleId": "e5f6g7h8-...",                                    │
│      "ruleType": "SEGMENT",                                      │
│      "evaluatedAt": "2026-04-08T10:00:00Z"                       │
│    }                                                               │
│  ]                                                                 │
│                                                                    │
│  Written by:   cms-runtime (EvaluateAll / EvaluatePlacement)      │
│  Read by:      cms-delivery (GetContentByPlacements)              │
│  Invalidated:  TTL expiry | schedule event | manual flush          │
└──────────────────────────────────────────────────────────────────┘
```

#### 7. Wiring — main.go additions

```go
// In cmd/server/main.go — after existing setup:

// 1. Create evaluator registry
evalRegistry := evaluator.NewRegistry()
evalRegistry.Register(&evaluator.ScoringEvaluator{})
evalRegistry.Register(&evaluator.SegmentEvaluator{})
evalRegistry.Register(&evaluator.EligibleEvaluator{})

// 2. Create cms-runtime service
runtimeSvc := service.NewCMSRuntimeService(
    scheduleRepo,
    redisRepo,    // CacheRepository
    postgresRepo, // DatabaseRepository
    evalRegistry,
    5*time.Minute, // ticker interval
    5*time.Minute, // cache TTL
    10,            // top N results per placement
)

// 3. Start the background evaluator
runtimeSvc.Start(context.Background())
defer runtimeSvc.Stop()

// 4. Create cms-delivery service
deliverySvc := service.NewCMSDeliveryService(redisRepo, 5*time.Minute)

// 5. Create and register delivery handler
deliveryHandler := handler.NewCMSDeliveryHandler(deliverySvc)
router.GET("/delivery", deliveryHandler.GetContent)
router.POST("/delivery/cache/flush", deliveryHandler.FlushCache)
```

#### 8. Reactive Hook — Schedule Service Integration

```go
// In existing ScheduleService.CreateSchedule() — add after successful persist:
func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *entity.Schedule) error {
    // ... existing overlap validation + persist ...

    // Trigger reactive re-evaluation for affected placement
    if s.runtimeSvc != nil && schedule.Placement != nil {
        go s.runtimeSvc.EvaluatePlacement(ctx, schedule.Placement.Name)
    }
    return nil
}

// Same pattern for UpdateSchedule()
```

### Key Features to Test

| Feature                         | What to Validate                                       | Success Criteria                                       |
| ------------------------------- | ------------------------------------------------------ | ------------------------------------------------------ |
| **Rule evaluation correctness** | Strategy dispatches to correct evaluator per rule type | SCORING/SEGMENT/ELIGIBLE rules produce expected scores |
| **Top N ranking**               | Results sorted by score, limited to N                  | Highest-scoring rules appear first, count ≤ N          |
| **Redis write/read cycle**      | cms-runtime writes → cms-delivery reads same data      | Data round-trips through JSON without loss             |
| **Cache-through fallback**      | Cache miss → PostgreSQL → populate → return            | First request is slower, subsequent requests hit cache |
| **Ticker lifecycle**            | Start → ticks fire → Stop gracefully                   | No goroutine leaks, ticker stops cleanly               |
| **Reactive trigger**            | Schedule create → immediate re-evaluation              | Placement cache updated within ms of schedule save     |
| **Cache flush**                 | Selective flush by placement + full flush              | Keys removed, next read triggers cache populate        |
| **API contract**                | GET /delivery?placements=x,y returns expected JSON     | Response matches documented schema                     |

---

## ✅ TEST: Validate with Users

### Testing Plan

**Method: Assumption Testing** — Identify and validate the riskiest assumptions underlying the prototype before committing to implementation.

**Tested with:** Product owner / domain expert (Nrtdemo), against the prototype artifacts.

| #   | Assumption Tested                              | Method                                       |
| --- | ---------------------------------------------- | -------------------------------------------- |
| A1  | ContentPath is globally unique per rule        | Direct question to domain expert             |
| A2  | EvaluateAll processes all placements           | Direct question — scope of ticker            |
| A3  | Redis JSON string is sufficient data structure | Direct question — simplicity vs features     |
| A4  | Cache miss → PostgreSQL fallback needed        | Direct question — delivery behavior on empty |
| A5  | Top N is fixed at 10                           | Direct question — flexibility needs          |

### User Feedback

| #   | Assumption                           | Verdict                                                            | Impact on Prototype                                                                                                                                          |
| --- | ------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| A1  | ContentPath is unique per rule       | **INVALIDATED** — Same content_path can appear in multiple rules   | Deduplication needed: when multiple rules point to same AEM path, keep the highest-scoring entry. Redis JSON may contain duplicate paths otherwise.          |
| A2  | EvaluateAll processes ALL placements | **INVALIDATED** — Only active schedules within current time window | `EvaluateAll()` must filter: `WHERE is_active = true AND effective_from <= NOW() AND effective_until > NOW()`. Reduces unnecessary evaluation work.          |
| A3  | Redis JSON string is acceptable      | **VALIDATED** — Start simple with JSON                             | Keeps compatibility with existing `CacheRepository` (Get/Set/GetSet). Can migrate to sorted sets later if needed.                                            |
| A4  | Cache miss → PostgreSQL fallback     | **INVALIDATED** — Return empty if Redis has no data                | cms-delivery does NOT query PostgreSQL on miss. If Redis is empty, return `[]`. Only cms-runtime populates Redis. Simplifies delivery service significantly. |
| A5  | Top N fixed at 10                    | **INVALIDATED** — Configurable per placement                       | Need a placement-level configuration for max results. Could be stored in `Placement` entity or a config map.                                                 |

### Key Learnings

**Three prototype changes required:**

#### Change 1: Content Path Deduplication in cms-runtime

```go
// In evaluateAndCache(), after scoring all rules, before sorting:
// Deduplicate by content_path — keep highest score per unique path
seen := make(map[string]scored)
for _, r := range results {
    if existing, ok := seen[r.Result.ContentPath]; !ok || r.Score > existing.Score {
        seen[r.Result.ContentPath] = r
    }
}
// Rebuild results from deduplicated map
results = make([]scored, 0, len(seen))
for _, r := range seen {
    results = append(results, r)
}
```

#### Change 2: Time-Window Filtering in EvaluateAll

```go
// EvaluateAll must only process schedules active in the current time window.
// Requires a new repository method:

// ScheduleRepository (new method):
ListActiveSchedulesInWindow(ctx context.Context, at time.Time) ([]*entity.Schedule, error)

// SQL equivalent:
// SELECT s.*, dr.*, p.*
// FROM schedules s
// JOIN decision_rules dr ON s.decision_rule_id = dr.id
// JOIN placements p ON s.placement_id = p.id
// WHERE s.is_active = true
//   AND s.deleted_at IS NULL
//   AND s.effective_from <= $1
//   AND s.effective_until > $1
//   AND dr.status = 'ACTIVE'
```

#### Change 3: Simplify cms-delivery — No PostgreSQL Fallback

```go
// cms-delivery becomes pure Redis read — no GetSet, no loader, no PG dependency

func (s *cmsDeliveryService) GetContentByPlacements(
    ctx context.Context,
    placements []string,
) (map[string][]ContentResult, error) {
    result := make(map[string][]ContentResult, len(placements))

    for _, placement := range placements {
        data, err := s.cacheRepo.Get(ctx, "placement:"+placement)
        if err != nil || data == "" {
            result[placement] = []ContentResult{} // empty, not an error
            continue
        }

        var items []ContentResult
        if err := json.Unmarshal([]byte(data), &items); err != nil {
            result[placement] = []ContentResult{}
            continue
        }
        result[placement] = items
    }
    return result, nil
}
```

**Impact:** cms-delivery no longer depends on `DatabaseRepository` — it only needs `CacheRepository`. This makes it a pure cache reader, which is cleaner for future microservice extraction.

#### Change 4: Configurable Top N per Placement

```go
// Option A: Add MaxResults field to Placement entity
type Placement struct {
    BaseModel
    Name        string `gorm:"size:255"  json:"name"`
    Description string `gorm:"type:text" json:"description"`
    MaxResults  int    `gorm:"default:10" json:"maxResults"`
}

// Option B: Configuration map (no entity change)
// topNConfig map[string]int = {"wsaHomeBanner": 5, "wsaSplash": 3, ...}
// Default to 10 if not specified
```

### Revised Architecture Flow (Post-Testing)

```
┌──────────────────────────────────────────────────────────────────┐
│                    cms-runtime (Background - REVISED)             │
│                                                                    │
│  Internal Go Ticker ──→ ListActiveSchedulesInWindow(now)          │
│          │                  (only active + within time window)     │
│          │                                                         │
│          ├──→ For each placement:                                  │
│          │      1. Find matching DecisionRules (status = ACTIVE)   │
│          │      2. Evaluate via Strategy (SCORING/SEGMENT/ELIGIBLE)│
│          │      3. Deduplicate by content_path (keep highest score)│
│          │      4. Score & rank → Top N (configurable per place.)  │
│          │      5. Write JSON → Redis  placement:{name}  + TTL    │
│          │                                                         │
│  Schedule create/update ──→ Trigger EvaluatePlacement(name)       │
└──────────────────────────────────────────────────────────────────┘
                              │
                         Redis (bridge)
                    placement:{name} → JSON
                              │
┌──────────────────────────────────────────────────────────────────┐
│              cms-delivery-service (HTTP - REVISED)                 │
│                                                                    │
│  GET /delivery?placements=wsaHomeBanner,wsaSplash                 │
│          │                                                         │
│          ├──→ Redis GET placement:{name}                          │
│          │      ├─ HIT  → parse JSON → return ranked content paths│
│          │      └─ MISS → return empty []  (NO PG fallback)       │
│          │                                                         │
│  POST /delivery/cache/flush?placements=...                        │
│                                                                    │
│  Dependencies: CacheRepository ONLY (no DatabaseRepository)       │
└──────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Next Steps

### Refinements Needed

Based on the testing phase, these refinements are required before implementation:

| #   | Refinement                                                                                           | Priority | Effort |
| --- | ---------------------------------------------------------------------------------------------------- | -------- | ------ |
| R1  | Add `ListActiveSchedulesInWindow(ctx, time.Time)` to `ScheduleRepository` interface + implementation | High     | Small  |
| R2  | Add content path deduplication logic in `evaluateAndCache()`                                         | High     | Small  |
| R3  | Add `MaxResults` field to `Placement` entity (or config map)                                         | Medium   | Small  |
| R4  | Remove `DatabaseRepository` dependency from `cmsDeliveryService` — pure cache read                   | High     | Small  |
| R5  | Add `Delete(ctx, key)` method to `CacheRepository` for selective cache flush                         | Medium   | Small  |
| R6  | Preload `DecisionRule` and `Placement` in `ListActiveSchedulesInWindow` query                        | High     | Small  |

### Action Items

#### Epic 1: cms-runtime — Rule Evaluation Engine

| Story   | Description                                                                              | Dependencies       |
| ------- | ---------------------------------------------------------------------------------------- | ------------------ |
| **1.1** | Create `RuntimeService` interface in `internal/domain/service/`                          | None               |
| **1.2** | Create `RuleEvaluator` interface + Registry in `internal/service/evaluator/`             | None               |
| **1.3** | Implement `ScoringEvaluator`, `SegmentEvaluator`, `EligibleEvaluator`                    | 1.2                |
| **1.4** | Add `ListActiveSchedulesInWindow()` to `ScheduleRepository` interface + PostgreSQL impl  | None               |
| **1.5** | Implement `cmsRuntimeService` with ticker, evaluation loop, deduplication, Top N ranking | 1.1, 1.2, 1.3, 1.4 |
| **1.6** | Add reactive hook: `ScheduleService.Create/Update` → `EvaluatePlacement()`               | 1.5                |
| **1.7** | Wire cms-runtime in `main.go` — registry setup, service creation, `Start()`/`Stop()`     | 1.5                |
| **1.8** | Unit tests for evaluators, deduplication, Top N ranking, time-window filtering           | 1.5                |

#### Epic 2: cms-delivery-service — Content Delivery

| Story   | Description                                                                               | Dependencies |
| ------- | ----------------------------------------------------------------------------------------- | ------------ |
| **2.1** | Create `DeliveryService` interface + `ContentResult` struct in `internal/domain/service/` | None         |
| **2.2** | Implement `cmsDeliveryService` — pure Redis read, return `[]` on miss                     | 2.1          |
| **2.3** | Add `Delete()` to `CacheRepository` interface + Redis impl for selective flush            | None         |
| **2.4** | Implement `FlushCache()` — selective by placement + full flush                            | 2.2, 2.3     |
| **2.5** | Create `CMSDeliveryHandler` — `GET /delivery` + `POST /delivery/cache/flush`              | 2.2, 2.4     |
| **2.6** | Wire delivery handler in `main.go` — register routes                                      | 2.5          |
| **2.7** | Unit tests for delivery service (cache hit, cache miss → empty, flush)                    | 2.2          |

#### Epic 3: Entity + Repository Changes

| Story   | Description                                                    | Dependencies |
| ------- | -------------------------------------------------------------- | ------------ |
| **3.1** | Add `MaxResults` field to `Placement` entity + migration       | None         |
| **3.2** | Create database migration for `MaxResults` column (default 10) | 3.1          |

#### Suggested Implementation Order

```
Phase 1 — Foundation (parallel tracks):
  ├─ Track A: 1.1, 1.2, 1.3 (interfaces + evaluators)
  ├─ Track B: 1.4, 3.1, 3.2 (repo + entity changes)
  └─ Track C: 2.1, 2.3 (delivery interface + cache delete)

Phase 2 — Core Services:
  ├─ 1.5 (cms-runtime service)
  └─ 2.2, 2.4 (cms-delivery service)

Phase 3 — HTTP + Wiring:
  ├─ 2.5, 2.6 (delivery handler + routes)
  ├─ 1.6 (reactive hooks)
  └─ 1.7 (main.go wiring)

Phase 4 — Testing:
  ├─ 1.8 (runtime unit tests)
  └─ 2.7 (delivery unit tests)
```

### Success Metrics

| Metric                      | Target                                     | How to Measure                                       |
| --------------------------- | ------------------------------------------ | ---------------------------------------------------- |
| **Cache hit rate**          | >95% during steady state                   | Redis `INFO stats` — keyspace_hits / (hits + misses) |
| **Delivery latency**        | <200ms p99                                 | Prometheus histogram on `GET /delivery`              |
| **Evaluation freshness**    | <ticker interval + 10s                     | Compare `evaluatedAt` timestamp vs current time      |
| **Placement coverage**      | 100% of active placements cached           | Count `placement:*` keys vs active placement count   |
| **Zero downtime**           | Ticker start/stop without goroutine leaks  | `runtime.NumGoroutine()` before/after lifecycle      |
| **API contract compliance** | Orchestrator receives expected JSON schema | Integration test with sample placements              |
| **Reactive latency**        | <1s from schedule save to cache update     | Log timestamp diff: schedule persist → Redis write   |

### Final Design Summary

```
┌─────────────────────────────────────────────────────────────────────┐
│                     KBank-ECMS-Backend Monolith                      │
│                                                                       │
│  ┌─────────────────────────┐    ┌──────────────────────────────┐    │
│  │    cms-runtime           │    │   cms-delivery-service        │    │
│  │    (Background)          │    │   (HTTP)                      │    │
│  │                          │    │                                │    │
│  │  ┌─ Go Ticker ──────┐   │    │   GET /delivery?placements=   │    │
│  │  │  Periodic eval    │   │    │         │                      │    │
│  │  └──────────────────┘   │    │         ├─ Redis HIT → JSON    │    │
│  │  ┌─ Reactive Hook ──┐   │    │         └─ Redis MISS → []     │    │
│  │  │  Schedule events  │   │    │                                │    │
│  │  └──────────────────┘   │    │   POST /delivery/cache/flush   │    │
│  │                          │    │                                │    │
│  │  Strategy Evaluators:    │    │   Deps: CacheRepository only  │    │
│  │  • ScoringEvaluator      │    └──────────────┬───────────────┘    │
│  │  • SegmentEvaluator      │                    │                    │
│  │  • EligibleEvaluator     │                    │                    │
│  │                          │                    │                    │
│  │  Deps: ScheduleRepo,    │                    │                    │
│  │        CacheRepo         │                    │                    │
│  └────────────┬─────────────┘                    │                    │
│               │                                   │                    │
│               ▼            Redis                  ▼                    │
│         ┌──────────────────────────────────┐                          │
│         │  placement:wsaHomeBanner → JSON   │                          │
│         │  placement:wsaPortBanner → JSON   │                          │
│         │  placement:wsaSplash     → JSON   │                          │
│         │  TTL: configurable                │                          │
│         └──────────────────────────────────┘                          │
│                                                                       │
│  ┌─── Shared Domain Layer ───────────────────────────────────┐       │
│  │  entity/: DecisionRule, Schedule, Placement               │       │
│  │  repository/: ScheduleRepository, CacheRepository         │       │
│  │  service/: RuntimeService, DeliveryService (interfaces)   │       │
│  └───────────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘
```

---

_Generated using BMAD Creative Intelligence Suite - Design Thinking Workflow_
