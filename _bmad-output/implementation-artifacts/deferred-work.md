# Deferred Work

## From: api-response-dto-wrapper (2026-04-07)

Pre-existing issues surfaced during adversarial review — not introduced by the `APIResponse` envelope change:

- **rule_management_handler: silently discards bind error** — `_ = c.ShouldBindJSON(&req)` ignores parse failures; the handler proceeds with a zero-value request.
- **DeleteSchedule: missing correlation headers on success** — `c.Status(http.StatusNoContent)` skips `setScheduleResponseHeaders`, so `Request-ID`, `Status-Code`, etc. are absent on 204 responses.
- **Internal Go error strings leaked to clients** — `err.Error()` is serialized directly into `APIResponse.Error`, exposing package names, field names, and type names to callers.
- **Duplicate Content-Type header** — `setScheduleResponseHeaders` sets `Content-Type: application/json; charset=UTF-8` (uppercase) while Gin's `c.JSON` also sets it (lowercase). Two conflicting headers are emitted on every response.
- **Service errors always return 422** — infrastructure failures (DB down, timeout) are returned as 422 Unprocessable Entity instead of 500 Internal Server Error.
- **Test helper `performRequest` ignores `http.NewRequest` error** — a construction failure causes a nil-dereference panic instead of a useful test failure.

## Deferred from: code review of 2-1-cms-runtime-service-interfaces (2026-04-08)

Pre-existing design decisions and implementation concerns surfaced during adversarial review — not in scope for this interfaces-only story:

- **Registry.Register nil guard** — calling `Register(nil)` panics on `e.RuleType()`; add a nil check or explicit panic message in the wiring layer (Story 2.2+).
- **Registry.Get empty string guard** — empty `ruleType` string silently returns `nil, false`; add guard or document the contract in the implementation (Story 2.2+).
- **NaN/Inf score propagation** — all three placeholder evaluators return `rule.Score` directly; `json.Marshal` of NaN/Inf returns an error that would corrupt the cache write. DB constraints prevent this in practice, but validate at cache-write layer in cmsRuntimeService (Story 2.2).
- **Context cancellation not pre-checked** — `EvaluateAll` and `EvaluatePlacement` implementations should check `ctx.Err()` before starting work; implement in Story 2.2+.
- **No input size limit on placementNames** — `GetContentByPlacements` accepts unbounded slice; enforce max size at HTTP handler/middleware level in Story 2.3.
- **RuntimeService Start/Stop idempotency** — calling `Start` twice or `Stop` before `Start` has undefined behavior; implementation must document and enforce lifecycle state machine in Story 2.2+.
