# Deferred Work

## From: api-response-dto-wrapper (2026-04-07)

Pre-existing issues surfaced during adversarial review — not introduced by the `APIResponse` envelope change:

- **rule_management_handler: silently discards bind error** — `_ = c.ShouldBindJSON(&req)` ignores parse failures; the handler proceeds with a zero-value request.
- **DeleteSchedule: missing correlation headers on success** — `c.Status(http.StatusNoContent)` skips `setScheduleResponseHeaders`, so `Request-ID`, `Status-Code`, etc. are absent on 204 responses.
- **Internal Go error strings leaked to clients** — `err.Error()` is serialized directly into `APIResponse.Error`, exposing package names, field names, and type names to callers.
- **Duplicate Content-Type header** — `setScheduleResponseHeaders` sets `Content-Type: application/json; charset=UTF-8` (uppercase) while Gin's `c.JSON` also sets it (lowercase). Two conflicting headers are emitted on every response.
- **Service errors always return 422** — infrastructure failures (DB down, timeout) are returned as 422 Unprocessable Entity instead of 500 Internal Server Error.
- **Test helper `performRequest` ignores `http.NewRequest` error** — a construction failure causes a nil-dereference panic instead of a useful test failure.
