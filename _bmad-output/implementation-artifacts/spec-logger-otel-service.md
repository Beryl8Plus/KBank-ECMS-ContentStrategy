---
title: 'Refactor logger to OTel-instrumented service with context propagation'
type: 'refactor'
created: '2026-04-19'
status: 'done'
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The logger package uses package-level functions with no context, making it impossible to link log events to distributed traces or export metrics — every service call is a black box in production observability tooling.

**Approach:** Add `context.Context` as the first parameter to all public logger functions; introduce `Init`/`Shutdown` that wire OTLP/gRPC trace and metric providers; each log call attaches a span event to the active span and increments level-scoped counters. All existing call sites are updated mechanically.

## Boundaries & Constraints

**Always:**
- All existing log format output (stdout/stderr lines) must be preserved exactly
- `context.Context` is the first parameter of every public log function
- OTel global providers (`otel.SetTracerProvider`, `otel.SetMeterProvider`) are used so the noop default works safely before `Init` is called
- Tracer name / instrumentation scope: `"kbank-ecms/logger"`
- Metric counters: `log_errors_total` (labels: `service`, `error_code`), `log_warnings_total` (labels: `service`)
- OTLP endpoint: read from `OTEL_EXPORTER_OTLP_ENDPOINT`; if empty, `Init` returns early with a noop setup (no error)
- `cmd/migrate` is a one-shot CLI — call `logger.Init` is skipped there; pass `context.Background()` to all logger calls
- `PopulateErrorLog` signature is unchanged (takes `*gin.Context`, does not log)
- Metric counters are created inside `Init` after the real MeterProvider is set; nil-checked in log functions so pre-`Init` calls are safe

**Ask First:**
- If any call site needs a non-`Background` context that cannot be sourced from the existing function parameters

**Never:**
- Do not remove or reformat existing log output lines
- Do not add dependency injection of a Logger struct to call sites — keep package-level functions
- Do not add OTel auto-instrumentation libraries beyond the core SDK and OTLP gRPC exporters

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| `Init` with no endpoint set | `OTEL_EXPORTER_OTLP_ENDPOINT=""` | Returns immediately, no-op OTel providers remain | No error returned |
| `LError` before `Init` | errCounter is nil | Logs to stderr normally; counter nil-check skips increment | Safe no-op |
| `LSystem(WARN)` before `Init` | warnCounter nil | Logs to stdout normally; counter nil-check skips increment | Safe no-op |
| `LError` with active span | span in ctx | Writes stderr log + records span event + sets span status Error + increments `log_errors_total` | N/A |
| `LSystem(WARN)` with active span | span in ctx | Writes stdout log + records span event + increments `log_warnings_total` | N/A |
| `Init` exporter dial fails | Unreachable OTLP endpoint | Exporter creation fails; `Init` returns the error to caller | Caller logs warning and continues without OTel |

</frozen-after-approval>

## Code Map

- `internal/infrastructure/logger/logger.go` — all public function signatures gain `ctx context.Context` first param; OTel span events and counter increments added
- `internal/infrastructure/logger/otel.go` — **new** — `Init(ctx, serviceName)` and package-level `errCounter`, `warnCounter` vars
- `internal/infrastructure/logger/logger_test.go` — update call sites to pass `context.Background()`
- `internal/grpc/middleware/interceptors.go` — pass `ctx` (already in scope) to `LError`, `LSystem`
- `internal/delivery/http/middleware/logger_middleware.go` — pass `c.Request.Context()` to `LRequest`
- `internal/infrastructure/database/database.go` — pass `ctx` (parameter) to `LStartup`
- `cmd/cms-runtime/main.go` — call `logger.Init`; pass `ctx`/`context.Background()` to logger calls
- `cmd/cms-delivery/main.go` — call `logger.Init`; pass `ctx`/`context.Background()` to logger calls
- `cmd/server/main.go` — call `logger.Init`; pass `ctx`/`context.Background()` to logger calls
- `cmd/migrate/main.go` — pass `context.Background()` only; no `Init`
- `go.mod` — add OTel SDK + OTLP gRPC exporter dependencies

## Tasks & Acceptance

**Execution:**
- [ ] `go.mod` -- add `go.opentelemetry.io/otel`, `/otel/sdk/trace`, `/otel/sdk/metric`, `/otel/exporters/otlp/otlptrace/otlptracegrpc`, `/otel/exporters/otlp/otlpmetric/otlpmetricgrpc` via `go get` then `go mod tidy`
- [ ] `internal/infrastructure/logger/otel.go` -- **create** — package-level `var errCounter metric.Int64Counter` and `var warnCounter metric.Int64Counter`; `Init(ctx context.Context, serviceName string) (func(context.Context) error, error)` that creates OTLP gRPC trace + metric exporters, builds TracerProvider and MeterProvider with `semconv.ServiceName(serviceName)` resource, sets globals via `otel.SetTracerProvider` / `otel.SetMeterProvider`, creates counters from the meter, returns shutdown func
- [ ] `internal/infrastructure/logger/logger.go` -- add `ctx context.Context` as first param to all public functions: `LError`, `LReqResClient`, `LRequest`, `LReqResApp`, `LResponse`, `LSystem`, `LStartup`; in each function: (a) before acquiring `mu`, call `trace.SpanFromContext(ctx).AddEvent(...)` with relevant fields as `attribute.String` pairs; (b) for `LError`: also call `span.SetStatus(codes.Error, e.Message)` and increment `errCounter` if non-nil; (c) for `LSystem` with `s.Level == "WARN"`: increment `warnCounter` if non-nil
- [ ] `internal/infrastructure/logger/logger_test.go` -- add `context.Background()` as first arg to every `LRequest` and `LError` call
- [ ] `internal/grpc/middleware/interceptors.go` -- add `ctx` (already the first param of `Log`) as first arg to `logger.LError` and `logger.LSystem`; add `ctx` to `recoveryHandler` via a closure over it or accept it as param — use `context.Background()` for panic recovery since original ctx may be poisoned
- [ ] `internal/delivery/http/middleware/logger_middleware.go` -- pass `c.Request.Context()` as first arg to `logger.LRequest`
- [ ] `internal/infrastructure/database/database.go` -- pass `ctx` (existing parameter) as first arg to `logger.LStartup`
- [ ] `cmd/cms-runtime/main.go` -- call `logger.Init(ctx, "cms-runtime")` after creating `ctx`; defer the returned shutdown; pass `ctx` or `context.Background()` to each `logger.*` call (use `context.Background()` for calls inside goroutines where original ctx may be cancelled)
- [ ] `cmd/cms-delivery/main.go` -- call `logger.Init(ctx, "cms-delivery")` after creating `ctx`; defer shutdown; update all call sites
- [ ] `cmd/server/main.go` -- call `logger.Init(ctx, "server")` after creating `ctx`; defer shutdown; update all call sites
- [ ] `cmd/migrate/main.go` -- no `Init`; pass `context.Background()` to all logger call sites

**Acceptance Criteria:**
- Given any logger call before `Init`, when the function is called, then the log line is written to stdout/stderr with the same format as before and no panic occurs
- Given `OTEL_EXPORTER_OTLP_ENDPOINT` is unset, when `Init` is called, then it returns `nil, nil` and subsequent log calls are safe
- Given an active OTel span in ctx, when `LError` is called, then a span event is recorded and span status is set to Error
- Given `LSystem` is called with `Level: "WARN"` after `Init`, when the call completes, then `log_warnings_total` counter is incremented with `service` label
- Given `LError` is called after `Init`, when the call completes, then `log_errors_total` is incremented with `service` and `error_code` labels
- Given the code is built, when `go build ./...` runs, then it exits 0

## Design Notes

**Span events vs. new spans:** Log calls attach events to the *existing active span* from ctx (`trace.SpanFromContext(ctx)`). They do NOT start new spans — that is the responsibility of middleware. If no span is active, `SpanFromContext` returns a noop span which is safe to call `AddEvent` on.

**Mutex scope:** OTel operations (`span.AddEvent`, counter `Add`) are goroutine-safe and must NOT be called while holding `mu`. Perform OTel operations first, then acquire the mutex only for the `fmt.Fprint` call.

**Recovery handler panic ctx:** In `recoveryHandler`, the original request context may be in an unknown state after a panic. Use `context.Background()` as the ctx for `logger.LError` in the panic recovery path.

**`Init` endpoint check:**
```go
endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
if endpoint == "" {
    return func(ctx context.Context) error { return nil }, nil
}
```

## Verification

**Commands:**
- `go build ./...` -- expected: exit 0
- `go vet ./internal/infrastructure/logger/...` -- expected: exit 0
- `go test ./internal/infrastructure/logger/...` -- expected: PASS
