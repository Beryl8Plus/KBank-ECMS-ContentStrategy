---
title: 'Install go-grpc-middleware and wire custom logger interceptors'
type: 'feature'
created: '2026-04-19'
status: 'done'
baseline_commit: 'feat/microservice-layer'
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The cms-runtime gRPC server has no request/response logging, no panic recovery, and no error tracing on the server side — making debugging and observability difficult in production.

**Approach:** Install `github.com/grpc-ecosystem/go-grpc-middleware/v2` and create a thin middleware package that adapts its logging and recovery interceptors to use the project's existing `internal/infrastructure/logger` functions, then wire them into the gRPC server in `cmd/cms-runtime/main.go`.

## Boundaries & Constraints

**Always:**
- All logging must go through `logger.LSystem`, `logger.LError`, or `logger.LStartup` — never raw `fmt.Print` or a third-party structured logger directly
- Interceptors (unary + stream) must both be registered
- Correlation ID must be extracted from gRPC metadata key `requestID` when present, otherwise a new UUID is generated
- Recovery interceptor must return `codes.Internal` to the caller and log the panic via `logger.LError`

**Ask First:**
- If a timeout interceptor is desired in addition to logging/recovery

**Never:**
- Do not replace or modify existing HTTP middleware (Gin)
- Do not add database, Redis, or cache dependencies to the middleware package
- Do not use `go.uber.org/zap`, `logrus`, or any other logger — only the project's custom logger

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Normal call | Valid `EvaluateRequest` | System log with method, duration, `OK` status | N/A |
| gRPC error returned | Handler returns `status.Error(codes.InvalidArgument, ...)` | System log with method, duration, error code/message | Log error; propagate original gRPC status |
| Panic in handler | Handler calls `panic("unexpected")` | `LError` log with stack trace; caller receives `codes.Internal` | Recovery interceptor catches; logs via `LError`; returns safe gRPC status |
| Missing correlation ID | No `requestID` in metadata | Auto-generated UUID used as `CorrelationID` | N/A |

</frozen-after-approval>

## Code Map

- `cmd/cms-runtime/main.go` -- gRPC server startup; wire `ChainUnaryInterceptor` / `ChainStreamInterceptor`
- `internal/grpc/server/runtime_grpc_server.go` -- existing handler; unchanged
- `internal/grpc/middleware/interceptors.go` -- **new** — custom `logging.Logger` adapter + recovery handler using project logger
- `internal/infrastructure/logger/logger.go` -- `LSystem`, `LError` functions consumed by middleware
- `internal/domain/entity/log.go` -- `SystemLog`, `ErrorLog` structs used to populate log calls
- `go.mod` -- add `github.com/grpc-ecosystem/go-grpc-middleware/v2`

## Tasks & Acceptance

**Execution:**
- [x] `go.mod` / `go.sum` -- run `go get github.com/grpc-ecosystem/go-grpc-middleware/v2` to add the dependency
- [x] `internal/grpc/middleware/interceptors.go` -- **create** — implement `grpcLogger` type satisfying `logging.Logger`; `LoggerUnaryInterceptor()` and `LoggerStreamInterceptor()` returning configured logging interceptors; `RecoveryUnaryInterceptor()` and `RecoveryStreamInterceptor()` using `logger.LError` for the panic handler
- [x] `cmd/cms-runtime/main.go` -- replace `grpc.NewServer()` with `grpc.NewServer(grpc.ChainUnaryInterceptor(...), grpc.ChainStreamInterceptor(...))` wiring both logging and recovery interceptors

**Acceptance Criteria:**
- Given the cms-runtime gRPC server is running, when a valid `EvaluateRequest` is received, then a system log line containing the gRPC method name and `OK` is written to stdout
- Given the cms-runtime gRPC server is running, when a handler returns a gRPC error, then a system log containing the error code and message is written
- Given the cms-runtime gRPC server is running, when a handler panics, then an error log is written via `LError` with a stack trace and the caller receives `codes.Internal`
- Given no `requestID` in gRPC metadata, when any call arrives, then a UUID is auto-generated and used as `CorrelationID` in log entries
- Given the code is built, when `go build ./...` runs, then it exits with code 0

## Design Notes

**Logger adapter pattern** — `go-grpc-middleware/v2/interceptors/logging` requires an interface:
```go
type Logger interface {
    Log(ctx context.Context, level logging.Level, msg string, fields ...any)
}
```
The adapter maps `logging.LevelInfo` → `logger.LSystem` with `Level: "INFO"` and `logging.LevelError` / `logging.LevelWarn` → `logger.LSystem` with the corresponding level string. Fields emitted by the middleware (`grpc.method`, `grpc.code`, `grpc.time_ms`) are extracted from the variadic `fields` key-value pairs and appended to `SystemLog.Message`.

**Correlation ID extraction** — use `google.golang.org/grpc/metadata` to extract `requestID` from incoming context at the start of each interceptor call.

## Verification

**Commands:**
- `go build ./...` -- expected: exit 0, no errors
- `go vet ./internal/grpc/middleware/...` -- expected: exit 0

## Suggested Review Order

**Entry point — server wiring**

- Recovery registered outermost so panics are caught before logging runs.
  [`main.go:42`](../../cmd/cms-runtime/main.go#L42)

**Interceptor implementation**

- Logger adapter: maps middleware levels to `LSystem`/`LError`; extracts correlation ID from metadata.
  [`interceptors.go:33`](../../internal/grpc/middleware/interceptors.go#L33)

- Correlation ID helper: reads `requestID` metadata key, falls back to UUID.
  [`interceptors.go:71`](../../internal/grpc/middleware/interceptors.go#L71)

- Recovery handler: captures panic, writes `LError` with stack trace, returns `codes.Internal`.
  [`interceptors.go:82`](../../internal/grpc/middleware/interceptors.go#L82)

**Public API surface**

- Four exported constructor functions (unary+stream for logging and recovery).
  [`interceptors.go:90`](../../internal/grpc/middleware/interceptors.go#L90)

**Supporting / peripherals**

- Dependency added at v2.3.3.
  [`go.mod:6`](../../go.mod#L6)
