# Story 2.3: CMS Service Implementations and Pod Entry Points

Status: done

## Story

As a **backend developer**,
I want to **implement `CMSRuntimeService` and `CMSDeliveryService` in `internal/service/`, and create two independent `cmd/` entry points (`cmd/cms-runtime` and `cmd/cms-delivery`) that run as separate pods**,
so that **the cms-runtime pod continuously evaluates active decision-rule schedules and writes results to Redis, and the cms-delivery pod serves those cached results via HTTP**.

## Acceptance Criteria

1. **AC-1: `CMSRuntimeService` implements `domainservice.RuntimeService`**
   - Given `internal/service/cms_runtime_service.go`
   - Then `CMSRuntimeService` struct has dependencies: `scheduleRepo ScheduleRepository`, `cacheRepo CacheRepository`, `registry *evaluator.Registry`, `resultTTL time.Duration`, `tickInterval time.Duration`
   - And `EvaluateAll(ctx)` calls `scheduleRepo.ListActiveSchedulesInWindow(ctx, time.Now())`, evaluates each schedule's DecisionRule via the registry, groups by `schedule.Placement.Name`, sorts by score descending, caps at `placement.MaxResults` (default 10 if 0), serialises to JSON, and writes to `cacheRepo.Set(ctx, "cms:placement:{name}", json, resultTTL)`
   - And `EvaluatePlacement(ctx, name)` calls `EvaluateAll` results filtered to the named placement
   - And `Start(ctx)` starts a background goroutine running `EvaluateAll` on the tick interval; returns error if already running
   - And `Stop()` signals the goroutine to stop and blocks until it exits; is a no-op if not running
   - And compile-time check `var _ domainservice.RuntimeService = (*CMSRuntimeService)(nil)` is present

2. **AC-2: `CMSDeliveryService` implements `domainservice.DeliveryService`**
   - Given `internal/service/cms_delivery_service.go`
   - Then `CMSDeliveryService` struct has dependency: `cacheRepo CacheRepository`
   - And `GetContentByPlacements(ctx, names)` reads `cacheRepo.Get(ctx, "cms:placement:{name}")` for each name, deserialises JSON `[]ContentResult`; on cache miss OR unmarshal error, includes an empty slice for that key
   - And nil/empty `names` returns an empty map with no error
   - And `FlushCache(ctx, names)` calls `cacheRepo.Delete` for each named placement key; if names is nil/empty, calls `cacheRepo.FlushDB(ctx)`
   - And compile-time check `var _ domainservice.DeliveryService = (*CMSDeliveryService)(nil)` is present

3. **AC-3: Cache key is consistent between runtime and delivery**
   - Given both services are in `package service`
   - Then a shared unexported function `cmsPlacementKey(name string) string` returns `"cms:placement:" + name`
   - And both `cms_runtime_service.go` and `cms_delivery_service.go` use this function

4. **AC-4: `cmd/cms-runtime/main.go` builds and wires the runtime pod**
   - Given `cmd/cms-runtime/main.go`
   - Then it connects to PostgreSQL and Redis using env vars
   - And wires `SchedulePostgresRepository`, `RedisRepository`, an `evaluator.Registry` (all 3 evaluators registered)
   - And creates `CMSRuntimeService` with `resultTTL=1h` default (env: `CMS_RUNTIME_TTL`) and `tickInterval=5m` default (env: `CMS_RUNTIME_INTERVAL`)
   - And calls `svc.Start(ctx)`, waits for `SIGINT/SIGTERM`, then calls `svc.Stop()`

5. **AC-5: `cmd/cms-delivery/main.go` and `cmd/cms-delivery/handler.go` build and wire the delivery pod**
   - Given `cmd/cms-delivery/main.go` and `cmd/cms-delivery/handler.go`
   - Then it connects to Redis only (no PostgreSQL) using env vars
   - And wires `RedisRepository` and `CMSDeliveryService`
   - And exposes `GET /content?placement=a&placement=b` → `GetContentByPlacements`
   - And exposes `POST /flush` → `FlushCache`
   - And listens on `PORT` env var (default `8082`)

6. **AC-6: `go build ./...` passes with zero errors**

7. **AC-7: Unit tests implemented and passing**
   - `internal/service/cms_runtime_service_test.go` — ≥5 test cases: EvaluateAll success, EvaluateAll with repo error, EvaluateAll respects MaxResults cap, EvaluatePlacement filters by name, Start/Stop lifecycle
   - `internal/service/cms_delivery_service_test.go` — ≥4 test cases: GetContentByPlacements cache hit, miss, nil input, FlushCache selective vs full
   - `cmd/cms-delivery/handler_test.go` — ≥4 test cases: GET /content OK, GET /content service error, POST /flush OK, POST /flush service error

## Tasks / Subtasks

- [x] **Task 1: Implement CMSRuntimeService** (AC: #1, #3, #6)
  - [x] 1.1 Create `internal/service/cms_runtime_service.go` with struct, constructor, and all 4 interface methods
  - [x] 1.2 Define unexported `cmsPlacementKey` helper in `cms_runtime_service.go`

- [x] **Task 2: Implement CMSDeliveryService** (AC: #2, #3, #6)
  - [x] 2.1 Create `internal/service/cms_delivery_service.go` with struct, constructor, and both interface methods

- [x] **Task 3: Bootstrap cmd/cms-runtime** (AC: #4, #6)
  - [x] 3.1 Create `cmd/cms-runtime/main.go`

- [x] **Task 4: Bootstrap cmd/cms-delivery** (AC: #5, #6)
  - [x] 4.1 Create `cmd/cms-delivery/handler.go` — `cmsDeliveryHandler` struct with `registerRoutes`, `getContent`, `flushCache`
  - [x] 4.2 Create `cmd/cms-delivery/main.go` — wire Redis, service, handler, gin router

- [x] **Task 5: Write unit tests** (AC: #7)
  - [x] 5.1 Create `internal/service/cms_runtime_service_test.go`
  - [x] 5.2 Create `internal/service/cms_delivery_service_test.go`
  - [x] 5.3 Create `cmd/cms-delivery/handler_test.go`

- [x] **Task 6: Verify build and full test suite** (AC: #6, #7)
  - [x] 6.1 Run `go build ./...` — zero errors
  - [x] 6.2 Run `go test ./internal/service/... ./cmd/cms-delivery/...`

## Dev Notes

### Source Reference

- Design Thinking Session: `_bmad-output/design-thinking-2026-04-08.md`
- Story 2-1: domain service interfaces (`cms_runtime.go`, `cms_delivery.go`, `evaluator/`)
- Story 2-2: `ListActiveSchedulesInWindow`, `CacheRepository.Delete`, `Placement.MaxResults`

### Architecture

```
cmd/cms-runtime/main.go
  └─ CMSRuntimeService (internal/service/cms_runtime_service.go)
       ├─ SchedulePostgresRepository — ListActiveSchedulesInWindow
       ├─ RedisRepository            — Set
       └─ evaluator.Registry         — ScoringEvaluator, SegmentEvaluator, EligibleEvaluator

cmd/cms-delivery/main.go
  └─ CMSDeliveryService (internal/service/cms_delivery_service.go)
       └─ RedisRepository            — Get, Delete, FlushDB
  └─ cmsDeliveryHandler (cmd/cms-delivery/handler.go)
       ├─ GET /content?placement=<name>
       └─ POST /flush
```

### Implementation Patterns (MUST FOLLOW)

**Service struct pattern** — follow `ScheduleService`:
```go
// CMSRuntimeService implements domainservice.RuntimeService.
type CMSRuntimeService struct {
    scheduleRepo domainrepo.ScheduleRepository
    cacheRepo    domainrepo.RedisCacheRepository
    registry     *evaluator.Registry
    resultTTL    time.Duration
    tickInterval time.Duration
    // lifecycle
    mu       sync.Mutex
    running  bool
    stopCh   chan struct{}
    done     chan struct{}
}

var _ domainservice.RuntimeService = (*CMSRuntimeService)(nil)
```

**Cache key** (unexported, shared via same package):
```go
func cmsPlacementKey(name string) string {
    return "cms:placement:" + name
}
```

**EvaluateAll core algorithm**:
```go
schedules, err := s.scheduleRepo.ListActiveSchedulesInWindow(ctx, time.Now())
// group by placement name
// for each group: evaluate via registry, sort score desc, cap at placement.MaxResults (default 10)
// marshal []ContentResult → JSON string → cacheRepo.Set(ctx, cmsPlacementKey(name), json, s.resultTTL)
```

**ContentResult fields**:
- `ContentPath` = `rule.ContentPath`
- `AemURL` = `rule.ContentPath` (AEM base URL not yet configured — same value, see deferred work)
- `Score` = score returned by evaluator
- `RuleID` = `rule.ID.String()`
- `RuleType` = `string(rule.Type)`
- `EvaluatedAt` = `time.Now().UTC().Format(time.RFC3339)`

**Start/Stop goroutine pattern**:
```go
func (s *CMSRuntimeService) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.running {
        return fmt.Errorf("cms-runtime: already running")
    }
    s.stopCh = make(chan struct{})
    s.done = make(chan struct{})
    s.running = true
    go s.runLoop(ctx)
    return nil
}

func (s *CMSRuntimeService) Stop() error {
    s.mu.Lock()
    if !s.running {
        s.mu.Unlock()
        return nil
    }
    close(s.stopCh)
    done := s.done
    s.mu.Unlock()
    <-done
    return nil
}
```

**cmd/cms-runtime** env vars:
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `CMS_RUNTIME_INTERVAL` (default `5m`)
- `CMS_RUNTIME_TTL` (default `1h`)

**cmd/cms-delivery** env vars:
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `PORT` (default `8082`)

**Handler struct in cmd/cms-delivery/handler.go** (package main):
```go
type cmsDeliveryHandler struct {
    svc domainservice.DeliveryService
}
```
Route `GET /content` reads `c.QueryArray("placement")`.
Route `POST /flush` binds `{"placements":["a","b"]}` — if body is absent/empty, passes nil.

**Imports** — use alias `domainservice "kbank-ecms/internal/domain/service"` throughout.

### Existing Files Referenced

- `internal/domain/service/cms_runtime.go` — RuntimeService interface
- `internal/domain/service/cms_delivery.go` — DeliveryService interface + ContentResult
- `internal/service/evaluator/evaluator.go` — Registry, RuleEvaluator
- `internal/service/evaluator/scoring_evaluator.go` — ScoringEvaluator (value receiver, stateless)
- `internal/repository/schedule_repository.go` — NewSchedulePostgresRepository
- `internal/repository/redis_repository.go` — NewRedisRepository
- `internal/infrastructure/database/` — NewPostgresDB
- `internal/domain/entity/` — Schedule, DecisionRule, Placement, RedisConfig, PostgresConfig
- `cmd/server/main.go` — reference for env-var wiring pattern

### Test Patterns (Follow schedule_service_test.go mock style)

```go
// Mock for CacheRepository — used in both runtime and delivery tests
type mockCacheRepository struct {
    getFn    func(ctx context.Context, key string) (string, error)
    setFn    func(ctx context.Context, key string, value string, exp time.Duration) error
    deleteFn func(ctx context.Context, key string) error
    flushFn  func(ctx context.Context) error
    // other methods: HGet, HSet, GetSet — no-op stubs
}
```

Handler tests use `httptest.NewRecorder()` + `httptest.NewRequest()` with a mock `domainservice.DeliveryService`.

### Module Name

`kbank-ecms` — all imports use this prefix.

---

## Dev Agent Record

### Implementation Plan

1. Implement `CMSRuntimeService` in `internal/service/cms_runtime_service.go`
2. Implement `CMSDeliveryService` in `internal/service/cms_delivery_service.go`
3. Create `cmd/cms-runtime/main.go` bootstrap pod
4. Create `cmd/cms-delivery/handler.go` + `cmd/cms-delivery/main.go` delivery pod
5. Write unit tests for all three packages
6. Verify `go build ./...` and `go test` pass

### Debug Log

- Fixed duplicate `package` declarations in `handler.go`, `cmd/cms-runtime/main.go`, and `handler_test.go` (artifact of file creation)
- `mockCacheRepo` in `pkg/util/cache_test.go` required a `Delete` stub after `CacheRepository.Delete` was added in Story 2-2

### Completion Notes

All 6 tasks complete. `go build ./...` clean. Tests:
- `internal/service`: 11 tests pass (5 CMS runtime, 5 CMS delivery, pre-existing schedule service tests)
- `cmd/cms-delivery`: 4 handler tests pass

Deferred: AemURL is set to `rule.ContentPath` as a placeholder (TODO: configure AEM base URL via env var)

---

## File List

- `internal/service/cms_runtime_service.go` — new
- `internal/service/cms_delivery_service.go` — new
- `internal/service/cms_runtime_service_test.go` — new
- `internal/service/cms_delivery_service_test.go` — new
- `cmd/cms-runtime/main.go` — new
- `cmd/cms-delivery/handler.go` — new
- `cmd/cms-delivery/main.go` — new
- `cmd/cms-delivery/handler_test.go` — new

---

## Change Log

| Date | Change |
|------|--------|
| 2026-04-08 | Story created — combined cms-runtime + cms-delivery service implementations and pod bootstraps |
