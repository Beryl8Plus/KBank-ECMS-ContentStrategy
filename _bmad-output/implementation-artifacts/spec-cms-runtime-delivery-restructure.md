---
title: "Restructure cms-runtime as pure gRPC evaluator, move orchestration to cms-delivery"
type: "refactor"
created: "2026-04-15"
status: "done"
baseline_commit: "833c167"
context:
  - "internal/grpc/server/runtime_grpc_server.go"
  - "internal/domain/service/cms_runtime_grpc.go"
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** cms-runtime currently bundles two concerns: (1) a background ticker that queries PostgreSQL, evaluates rules, and writes to Redis, and (2) a gRPC server for on-demand evaluation. This couples DB/cache access into a pod that should be a stateless compute service.

**Approach:** Strip cms-runtime down to a pure gRPC evaluation server (no DB, no Redis, no ticker, no CacheMemory, no evaluator.Registry). Remove the three separate evaluators (ScoringEvaluator, SegmentEvaluator, EligibleEvaluator) — they all return `rule.Score` when `userAttrs == nil`, making the registry indirection unnecessary. The gRPC server reads `rule.Score` directly, deduplicates, sorts, and returns. Move the background evaluation orchestration — cache memory (L1), cache Redis (L2/L3), and the periodic data-transfer loop — into cms-delivery, which already has gRPC + scheduleRepo; it will gain Start/Stop lifecycle and a ticker that sends schedules to cms-runtime via gRPC, then caches results locally.

## Boundaries & Constraints

**Always:**

- cms-runtime must remain a pure stateless gRPC server — no external dependencies (no Registry, no DB, no cache).
- cms-delivery must use its existing gRPC client (RuntimeEvaluator) to delegate evaluation to cms-runtime.
- Existing gRPC proto, RuntimeEvaluator interface, and gRPC server/client code must not change.
- Cache key formats (`cms:placement:logic:{name}`, `cms:rule_logic:v1:{hash}`) must stay the same.

**Ask First:**

- Removing or modifying the `RuntimeService` domain interface.
- Any changes to the gRPC proto definitions.

**Never:**

- Duplicate condition evaluation logic in cms-delivery (EvaluateLogicConditions stays in the evaluator package, called at delivery time).
- Break the existing GetContentByPlacements / GetPersonalizedContent HTTP paths.
- Remove graceful degradation — cms-delivery must still work when gRPC is unavailable (no ticker writes, but reads from existing cache continue).

## I/O & Edge-Case Matrix

| Scenario                  | Input / State                           | Expected Output / Behavior                                                | Error Handling                 |
| ------------------------- | --------------------------------------- | ------------------------------------------------------------------------- | ------------------------------ |
| Ticker fires              | Active schedules in DB, gRPC up         | Logic entries written to Redis per placement                              | Log and continue per placement |
| gRPC unavailable at tick  | Active schedules in DB, gRPC down       | Tick logged as warning, stale cache remains                               | No crash, retry next tick      |
| cms-runtime receives gRPC | EvaluateSchedulesRequest with schedules | Returns score-sorted ContentResult (direct rule.Score, no condition eval) | gRPC status error              |
| Shutdown signal           | SIGINT/SIGTERM                          | Ticker stops, gRPC graceful stop, clean exit                              | Log errors                     |

</frozen-after-approval>

## Code Map

- `cmd/cms-runtime/main.go` — entry point: strip DB/Redis/CacheMemory/ticker/evaluator.Registry, keep gRPC server only
- `cmd/cms-delivery/main.go` — entry point: add CacheMemory, ticker config, Start/Stop lifecycle
- `internal/service/cms_runtime_service.go` — remove entire file (gRPC server handles evaluation directly)
- `internal/service/cms_delivery_service.go` — add: CacheMemory field, background ticker (Start/Stop/runLoop), orchestration loop using gRPC
- `internal/domain/service/cms_runtime.go` — RuntimeService interface: remove (no implementors remain)
- `internal/grpc/server/runtime_grpc_server.go` — remove evaluator.Registry dependency; read rule.Score directly, use BuildPlacementLogicEntry for logic path
- `internal/service/evaluator/scoring_evaluator.go` — remove (unnecessary for gRPC path)
- `internal/service/evaluator/segment_evaluator.go` — remove (unnecessary for gRPC path)
- `internal/service/evaluator/eligible_evaluator.go` — remove (unnecessary for gRPC path)
- `internal/service/evaluator/evaluator.go` — remove Registry type (no consumers remain); keep package for BuildPlacementLogicEntry, EvaluateLogicConditions, GenerateConditionHash
- `internal/cms-delivery/handler/routes.go` — update NewCMSDeliveryService call signature if constructor changes

## Tasks & Acceptance

**Execution:**

- [x] `internal/service/evaluator/scoring_evaluator.go` — remove file
- [x] `internal/service/evaluator/segment_evaluator.go` — remove file
- [x] `internal/service/evaluator/eligible_evaluator.go` — remove file
- [x] `internal/service/evaluator/evaluator.go` — remove Registry type and Register/Get methods; keep package exports (BuildPlacementLogicEntry, EvaluateLogicConditions, GenerateConditionHash, EvaluateRuleScore)
- [x] `internal/grpc/server/runtime_grpc_server.go` — remove Registry field; read rule.Score directly in EvaluateSchedules; use evaluator.BuildPlacementLogicEntry with score in EvaluatePlacementLogic; remove NewRuntimeGRPCServer registry param
- [x] `internal/service/cms_runtime_service.go` — remove entire file
- [x] `internal/domain/service/cms_runtime.go` — remove RuntimeService interface
- [x] `cmd/cms-runtime/main.go` — remove PostgreSQL, Redis, CacheMemory, evaluator.Registry, CMSRuntimeService; retain only gRPC server (updated Register call) + signal handling
- [x] `internal/service/cms_delivery_service.go` — add `cacheMemory *cache.CacheMemory[*entity.DecisionRule]`, `tickInterval time.Duration`, `mu/running/stopCh/done` fields; add `Start(ctx)`, `Stop()`, `runLoop()`, `evaluateAllViaGRPC(ctx)` methods that query schedules, group by placement, delegate evaluation to gRPC, write L1/L2/L3 caches
- [x] `cmd/cms-delivery/main.go` — init CacheMemory, parse CMS_RUNTIME_TTL / CMS_RUNTIME_INTERVAL env vars, pass to NewCMSDeliveryService, call svc.Start/Stop with signal handling
- [x] `internal/cms-delivery/handler/routes.go` — update constructor call and route wiring to match new service signature

**Acceptance Criteria:**

- Given cms-runtime starts, when no DB/Redis env vars are set, then it boots with only gRPC listener (no Registry, no DB, no cache)
- Given cms-delivery starts with gRPC addr configured, when ticker fires, then it queries DB, calls gRPC, and writes logic entries to Redis
- Given cms-delivery ticker is running and gRPC is unavailable, when ticker fires, then it logs a warning and does not crash
- Given SIGINT sent to cms-delivery, when background ticker is running, then Stop() blocks until goroutine exits cleanly

## Verification

**Commands:**

- `go build ./cmd/cms-runtime` — expected: compiles without DB/Redis/cache/evaluator.Registry imports
- `go build ./cmd/cms-delivery` — expected: compiles with new ticker + CacheMemory deps
- `go vet ./...` — expected: no issues
