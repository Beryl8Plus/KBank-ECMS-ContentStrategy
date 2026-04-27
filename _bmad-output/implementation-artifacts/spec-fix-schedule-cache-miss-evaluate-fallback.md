---
title: 'Fix schedule cache miss: trigger evaluate and retry instead of silently skipping'
type: 'bugfix'
created: '2026-04-24'
status: 'done'
baseline_commit: '796a8d16f7650dda99b737865650e784d91fe96f'
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** When `GetPersonalizedContent` encounters a cache miss on `cacheMemory.Schedules` (e.g. after TTL expiry), it silently skips the affected placement and returns empty results instead of falling back to re-evaluating from the database.

**Approach:** On schedule cache miss, call `evaluate()` to re-populate all placement caches from the database, then retry reading the missed placements from cache before proceeding with evaluation. Also remove a duplicate code block (the missedRules + filter logic appears twice in the same function).

## Boundaries & Constraints

**Always:**
- Graceful degradation: if `evaluate()` also yields no data (e.g. memory pressure still active), the placement is silently skipped — no error is returned to the caller.
- `occurrenceRepo` nil check must guard the fallback (existing pattern).
- The retry loop must preserve correct order-independence — placements already found in cache on the first pass must not be re-fetched.

**Ask First:**
- If a singleflight / deduplicated evaluate call is needed to prevent thundering herd under load (out of scope now; background ticker normally prevents this; ask only if explicitly requested).

**Never:**
- Do not add network/DB calls inside the per-placement retry loop; re-use the single `evaluate()` call that already queries all active placements.
- Do not change the `CacheMemory.Set()` eviction logic under memory pressure.
- Do not add new fields to `CMSDeliveryService` struct.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Cache warm | Schedules found in `cacheMemory.Schedules` for all requested placements | Normal evaluation flow, `evaluate()` NOT called | — |
| Cache miss (TTL expired) | `cacheMemory.Schedules.Get()` misses for ≥1 placement, `occurrenceRepo.ListActiveAt` returns active schedules | `evaluate()` called once, missed placements retried from cache, non-empty results returned | — |
| Cache miss + no active occurrences | `ListActiveAt` returns empty | `evaluate()` called, retry yields miss, placement skipped, empty slice returned gracefully | No error propagated |
| Cache miss + `occurrenceRepo == nil` | `occurrenceRepo` is nil | No fallback attempted; placement skipped silently (same as before) | — |

</frozen-after-approval>

## Code Map

- `cmd/svc-contstrat-delivery/service/cms_delivery_service.go` -- `GetPersonalizedContent` (lines 166–309): contains the cache-miss bug (lines 191–205) and the duplicate missedRules+filter block (lines 251–280)
- `cmd/svc-contstrat-delivery/service/cms_delivery_service.go` -- `evaluate()` (lines 440–500): re-populates `cacheMemory.Schedules` for all active placements; called by background ticker and now also on cache miss
- `cmd/svc-contstrat-delivery/service/cms_delivery_service_test.go` -- existing test file for the service; new tests go here

## Tasks & Acceptance

**Execution:**
- [x] `cmd/svc-contstrat-delivery/service/cms_delivery_service.go` -- In `GetPersonalizedContent`, replace the silent-skip on schedule cache miss (lines 191–205) with: collect missed placement names → call `s.evaluate(ctx)` once if any miss and `s.occurrenceRepo != nil` → retry missed placements from cache
- [x] `cmd/svc-contstrat-delivery/service/cms_delivery_service.go` -- Remove the duplicate missedRules+filter block (lines 251–280 which are an exact duplicate of lines 209–249)
- [x] `cmd/svc-contstrat-delivery/service/cms_delivery_service_test.go` -- Add `TestGetPersonalizedContent_CacheMissTriggersEvaluate`: cache starts empty, `listActiveAtFn` returns one active schedule (with Placement+DecisionRule), verify `ListActiveAt` is called and result contains expected content
- [x] `cmd/svc-contstrat-delivery/service/cms_delivery_service_test.go` -- Add `TestGetPersonalizedContent_CacheMissPersistsGracefully`: cache starts empty, `listActiveAtFn` returns empty, verify result is empty slice with no error

**Acceptance Criteria:**
- Given schedule cache is empty (TTL expired), when `GetPersonalizedContent` is called, then `evaluate()` is triggered and results are returned from the freshly populated cache.
- Given `occurrenceRepo` is nil, when schedule cache misses, then the placement is skipped silently (no panic, no error).
- Given `ListActiveAt` returns no occurrences after evaluate, when cache retry still misses, then an empty result is returned without error.
- Given the duplicate missedRules+filter block is removed, when the function runs, then behavior is identical to before (only the first occurrence of the block is kept).

## Design Notes

The fallback evaluates ALL placements in one DB round-trip (`evaluate()` already does this) rather than querying per-placement. This is intentional: the background ticker uses the same function and the query cost is already accounted for. Calling `evaluate()` inline from the request path is safe because `evaluate()` is idempotent and uses the same mutex-free cache writes as the ticker.

```
Cache hit  → normal path, no change
Cache miss → evaluate() → retry from cache → evaluation
          ↓ (miss persists)
          → placement silently skipped (graceful degradation)
```

## Verification

**Commands:**
- `go test ./cmd/svc-contstrat-delivery/service/... -v -run TestGetPersonalizedContent` -- expected: all tests pass
- `go test ./cmd/svc-contstrat-delivery/service/... -race` -- expected: no race conditions detected
- `go build ./cmd/svc-contstrat-delivery/...` -- expected: build succeeds

## Suggested Review Order

**Cache-miss fallback logic (main fix)**

- Entry point: cache miss collected into map, evaluate() triggered once, retry loop
  [`cms_delivery_service.go:191`](../../../cmd/svc-contstrat-delivery/service/cms_delivery_service.go#L191)

- evaluate() — the function called on miss; populates cacheMemory.Schedules from DB
  [`cms_delivery_service.go:420`](../../../cmd/svc-contstrat-delivery/service/cms_delivery_service.go#L420)

**Duplicate block removal**

- Single surviving missedRules+filter block (duplicate removed above this line)
  [`cms_delivery_service.go:218`](../../../cmd/svc-contstrat-delivery/service/cms_delivery_service.go#L218)

**Tests**

- New test: cache miss triggers evaluate and returns results
  [`cms_delivery_service_test.go:395`](../../../cmd/svc-contstrat-delivery/service/cms_delivery_service_test.go#L395)

- New test: graceful empty result when evaluate finds no occurrences
  [`cms_delivery_service_test.go:450`](../../../cmd/svc-contstrat-delivery/service/cms_delivery_service_test.go#L450)

## Spec Change Log

