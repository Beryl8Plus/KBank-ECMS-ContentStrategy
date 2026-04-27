# Cache Invalidation on Decision-Rule Activation

## Overview

The CMS Delivery service (`svc-contstrat-delivery`, port 8082) keeps an L1 in-memory mirror of two cache families per placement:

- `schedules:placement:{placementName}` — the `[]*entity.Schedule` set used to evaluate content for a placement.
- `rule:{decision_uuid}` — the materialized `*entity.DecisionRule` referenced by those schedules.

When an operator activates a decision rule in the Backoffice service (`svc-contstrat-backoffice`, port 8081), the underlying schedule rows are flipped from `DRAFT` to `ACTIVE`, but every delivery pod still holds the previous evaluation in memory. Without an explicit signal, pods serve stale content until the next ticker refresh or the cache TTL expires.

This feature wires the activation endpoint to a Redis Pub/Sub publisher. On a successful activation, the backoffice broadcasts a structured `SyncPingMessage` for each placement that the rule touches. Each delivery pod receives the message, **manually removes** the two affected cache keys, and re-evaluates the placement so the next read returns the freshly-activated rule. **TTL** acts as a safety net if a ping is dropped.

---

## Trigger

| Layer          | File                                                                       | Symbol                                                                   |
| -------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| Route          | `cmd/svc-contstrat-backoffice/handler/routes.go:43`                        | `decisionRules.PUT("/:id/activate", wizardHandler.ActivateDecisionRule)` |
| Handler        | `cmd/svc-contstrat-backoffice/handler/decision_rule_wizard_handler.go:197` | `ActivateDecisionRule(c *gin.Context)`                                   |
| Service        | `internal/service/decision_rule_wizard_service.go:442`                     | `ActivateStep4(ctx, id)`                                                 |
| Publisher hook | `internal/service/decision_rule_wizard_service.go`                         | `invalidateCachesForActivation(ctx, id, schedules)`                      |

The publish call fires **after** `repo.ActivateDecisionRule` returns successfully and **before** the HTTP response is written. It is fire-and-forget: a publish failure is logged at `WARN` and never surfaces as an HTTP error — the activation has already been persisted, and stale entries fall out via TTL.

`ActivateStep4` already loads the schedules with `Placement` preloaded (`FindSchedulesByDecisionRuleID`), so determining the affected placement names requires no extra query. Distinct placement names are deduplicated before publishing so each placement receives exactly one ping per activation.

---

## Pub/Sub Topology

| Item               | Value                                                  |
| ------------------ | ------------------------------------------------------ |
| Channel            | `cms:sync:ping` (constant `pubsub.ChannelCMSSyncPing`) |
| Direction          | Backoffice → Delivery (1 publisher → N subscribers)    |
| Encoding           | JSON-marshaled `pubsub.SyncPingMessage`                |
| Delivery semantics | At-most-once (Redis Pub/Sub does not persist)          |

### Message: `SyncPingMessage`

Defined in `internal/infrastructure/pubsub/cms_sync.go`:

```go
type SyncPingMessage struct {
    PlacementName  string `json:"placement_name"`
    VersionHash    string `json:"version_hash"`
    DecisionRuleID string `json:"decision_rule_id,omitempty"`
}
```

| Field            | Meaning                                                                                                                                                                                                                                                                                                                          |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PlacementName`  | Scopes the invalidation to one placement. Empty → subscriber falls back to a full evaluate of all placements.                                                                                                                                                                                                                    |
| `VersionHash`    | When non-empty, subscribers skip the refresh if their local version already matches. **Activate pings send `""`** to force a refresh regardless of local state.                                                                                                                                                                  |
| `DecisionRuleID` | When set, subscribers explicitly `Delete` the `rule:{id}` and `schedules:placement:{name}` cache entries before re-evaluating. **Optional and backwards compatible** — older publishers omit the field, and the subscriber simply skips the manual delete and relies on `UpdateSchedules` to reconcile during the next evaluate. |

### Channel constant

The literal `"cms:sync:ping"` is defined exactly once, in the shared package:

```go
const ChannelCMSSyncPing = "cms:sync:ping"
```

Both backoffice and delivery import this constant — they cannot drift.

---

## Cache Keys

| Key                                   | Cache                      | Type                                       | TTL                               | Invalidation triggers                                                                                  |
| ------------------------------------- | -------------------------- | ------------------------------------------ | --------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `schedules:placement:{placementName}` | `MemoryCache.Schedules`    | `*cache.CacheMemory[[]*entity.Schedule]`   | `CMS_RUNTIME_TTL` (default `15m`) | Manual via subscriber on ping + TTL fallback + `UpdateSchedules` reconciliation on each evaluate       |
| `rule:{decision_uuid}`                | `MemoryCache.DecisionRule` | `*cache.CacheMemory[*entity.DecisionRule]` | `CMS_RUNTIME_TTL` (default `15m`) | Manual via subscriber on ping + TTL fallback + `UpdateSchedules` reconciliation + `PruneOrphanedRules` |

### Invalidation actions

#### 1. Manual remove (primary path for this feature)

When a `SyncPingMessage` arrives with `DecisionRuleID != ""`, the subscriber, after the version-hash short-circuit and jitter delay, calls:

```go
s.cacheMemory.DecisionRule.Delete(ruleDecisionCacheKey(ping.DecisionRuleID))
s.cacheMemory.Schedules.Delete(cmsPlacementSchedulesKey(ping.PlacementName))
```

`CacheMemory.Delete` (defined at `internal/infrastructure/cache/memory.go:131`) removes the key from the underlying `patrickmn/go-cache` store and refreshes Prometheus metrics. After the delete, the subscriber calls `s.evaluate(ctx, ping.PlacementName)`, which queries Postgres for the active occurrences and rewrites both cache families via `UpdateSchedules`.

This guarantees that **between the delete and the rewrite, no reader can observe a stale `rule:{id}` mirror** — even briefly.

#### 2. TTL (safety net)

Every `CacheMemory.Set(key, value, ttl)` honors the configured TTL (`CMS_RUNTIME_TTL`, default `15m`). If a ping is dropped — Redis outage, slow consumer dropping the buffered message, or a pod that booted after the publish — the next read after TTL expiry forces a fresh evaluate from Postgres. TTL guarantees eventual consistency without requiring at-least-once delivery from Pub/Sub.

---

## Sequence

```
Operator                Backoffice                  Redis Pub/Sub             Delivery Pod (×N)
   |                        |                            |                            |
   |---PUT /activate------->|                            |                            |
   |                        |--ActivateStep4 (DB write)->|                            |
   |                        |   FindSchedulesByDecisionRuleID                         |
   |                        |   ActivateDecisionRule                                  |
   |                        |--PUBLISH cms:sync:ping---->|                            |
   |                        |   {placement_name=A,       |                            |
   |                        |    decision_rule_id=R,     |                            |
   |                        |    version_hash=""}        |                            |
   |                        |--PUBLISH cms:sync:ping---->|                            |
   |                        |   {placement_name=B, ...}  |                            |
   |                        |                            |---fanout------------------>| (each pod)
   |<--200 OK---------------|                            |                            |
   |                        |                            |                            |--SyncPingMessage
   |                        |                            |                            |  unmarshal
   |                        |                            |                            |  jitter (50–500ms)
   |                        |                            |                            |  Delete rule:R
   |                        |                            |                            |  Delete schedules:placement:A
   |                        |                            |                            |  evaluate(A) → DB
   |                        |                            |                            |  UpdateSchedules → repopulate
```

---

## Failure Modes

| Failure                                               | Behavior                                                                                                                                                                                                                                                                                                                                 |
| ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Backoffice cannot connect to Redis at startup         | `main.go` logs `ERROR`, leaves `redisCache` as nil interface, `pubsub.NewPublisher(nil)` returns a no-op publisher. `PingPlacement` returns `nil` immediately. Activations succeed; cache invalidation reverts to TTL only.                                                                                                              |
| Publish fails at request time                         | Service logs `WARN` (`DECISION-RULE-WIZARD: activate: cache-invalidation ping failed for placement=...`) and continues. HTTP response stays `200`.                                                                                                                                                                                       |
| Subscriber dropped the message (consumer buffer full) | Logged in `RedisRepository.Subscribe` as `WARN: pubsub: message dropped on channel "cms:sync:ping" — consumer buffer full`. Cache for that placement falls back to TTL expiry.                                                                                                                                                           |
| Subscriber receives malformed JSON                    | `subscribeToUpdates` falls through to a full evaluate (existing behavior at `cms_delivery_service.go:840-845`). Safe but more expensive than a targeted refresh.                                                                                                                                                                         |
| Two pods both receive the same ping                   | Each independently runs the version-hash short-circuit. After jitter, both delete and re-evaluate. The duplicate work is bounded; subsequent identical pings (same `version_hash`) are skipped. Activate pings carry `version_hash=""` so this dedupe does not apply — but jitter (50–500ms) flattens the load curve across the cluster. |
| `cacheMemory == nil` on the subscriber pod            | All `Delete` calls are guarded by `if s.cacheMemory != nil`; the pod skips local invalidation and the next evaluate runs unguarded.                                                                                                                                                                                                      |

---

## Testing

### Unit

`internal/service/decision_rule_wizard_activate_test.go` provides three table-style tests:

1. `TestActivateStep4_PublishesOnePingPerPlacement` — three schedules across two placements (one duplicate); asserts exactly two publishes, both on `pubsub.ChannelCMSSyncPing`, each carrying the rule UUID and an empty version hash.
2. `TestActivateStep4_NilPublisherIsNoOp` — verifies that activations work end-to-end when the publisher is nil (mirrors the production failure-soft behavior when Redis init fails).
3. `TestActivateStep4_PublishErrorDoesNotFailActivation` — recording publisher returns an error; `ActivateStep4` still returns `ACTIVE` and `200`.

Run with race detection:

```bash
go test ./internal/service/ -run "TestActivateStep4" -race
```

### Integration / manual

1. `make dev-up` (Postgres + Redis), `make migrate`, seed a decision rule with two schedules across two placements.
2. Start delivery: `go run ./cmd/svc-contstrat-delivery/`. Confirm log line `subscriber: listening on "cms:sync:ping"`.
3. Hit `GET /content?placement=<name>` once on delivery to warm the cache.
4. Start backoffice: `SETENV=DEVLOCAL REDIS_HOST=localhost REDIS_PORT=6379 go run ./cmd/svc-contstrat-backoffice/`.
5. `PUT /decision-rules/{id}/activate` — confirm:
   - Backoffice log: `publisher: ping sent for placement="<name>" rule="<uuid>"` per affected placement.
   - Delivery log: `subscriber: received ping for "<name>" (version ), triggering evaluate`.
   - The next `GET /content` returns the newly-activated rule's content.

---

## Configuration

| Env var           | Default | Purpose                                                                                              |
| ----------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `REDIS_HOST`      | —       | Redis host (required for publish to function; nil-safe if absent)                                    |
| `REDIS_PORT`      | —       | Redis port                                                                                           |
| `REDIS_PASSWORD`  | —       | Redis password (or use Workload Identity in non-DEVLOCAL)                                            |
| `CMS_RUNTIME_TTL` | `15m`   | TTL for `MemoryCache.Schedules` and `MemoryCache.DecisionRule` entries — the safety-net invalidation |

---

## Related Code

| Component                                          | File:line                                                                                             |
| -------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Channel constant + `SyncPingMessage` + `Publisher` | `internal/infrastructure/pubsub/cms_sync.go`                                                          |
| `Publish` interface method                         | `internal/domain/repository/redis_cache.go`                                                           |
| `Publish` implementation (go-redis)                | `internal/repository/redis_repository.go`                                                             |
| Activate handler                                   | `cmd/svc-contstrat-backoffice/handler/decision_rule_wizard_handler.go:197`                            |
| Activate service + invalidation hook               | `internal/service/decision_rule_wizard_service.go` (`ActivateStep4`, `invalidateCachesForActivation`) |
| Backoffice DI wiring                               | `cmd/svc-contstrat-backoffice/providers.go`, `wire.go`, `main.go`                                     |
| Subscriber loop                                    | `cmd/svc-contstrat-delivery/service/cms_delivery_service.go:803-905` (`subscribeToUpdates`)           |
| Cache primitive `Delete`                           | `internal/infrastructure/cache/memory.go:131`                                                         |
| Reconciling reload after delete                    | `cms_delivery_service.go` (`UpdateSchedules`)                                                         |
