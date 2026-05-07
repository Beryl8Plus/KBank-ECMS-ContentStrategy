# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

KBank ECMS CMS Delivery Runtime Service — a read-only, high-throughput personalization API that evaluates campaign content rules against user attributes and returns ranked content results. Module name: `kbank-ecms`.

Server listens on port **8082** by default. Swagger UI at `/swagger/index.html`, Prometheus metrics at `/metrics`, health check at `/healthz`.

## Commands

```bash
# Initial setup (installs golangci-lint, swag, wire, git hooks)
make init

# Start infrastructure (PostgreSQL + Redis)
make dev-up && make dev-down

# Run locally
make run

# Build binary (also regenerates Swagger docs)
make build

# Regenerate Swagger docs only
make swagger

# Format code (go fmt + custom GORM/JSON tag aligner via scripts/format_tags.go)
make fmt

# Lint
make lint

# All tests
make test
go test ./...

# Run a specific test
go test ./cmd/server/service/... -run TestCMSDeliveryService

# Tests with race detector
go test -race ./...
```

## Architecture

### Request Flow

```
GET /api/content-strategy/v1/personalized-content
  └── Gin router (internal/delivery/http/router.go)
      └── Middleware: auth · rate-limit · timeout · CORS · Prometheus
          └── Handler (cmd/server/handler/)
              └── CMSDeliveryService.GetPersonalizedContent()
                  └── L1 MemoryCache → L2 Redis → L3 PostgreSQL (fallback)
                      └── LocalEvaluator.Evaluate() → []dto.ContentResult
```

### Cache Architecture (Three Layers)

- **L1**: Per-pod `MemoryCache` (patrickmn/go-cache) — keyed by `schedules:placement:{name}` and `rule:{id}`
- **L2**: Redis (go-redis v9) — distributed shared cache
- **L3**: PostgreSQL via GORM — source-of-truth fallback

**Invalidation**: Redis Pub/Sub on `cms:sync:ping` channel. Pods apply 50–500 ms jitter before re-evaluating to prevent stampede. Subscribers check local version hash first and skip pulls if already up-to-date.

**Background ticker**: `runLoop` fires `evaluate()` every `CMS_RUNTIME_INTERVAL` (default 5 m) to warm all placement caches. `warmupCache` runs at startup.

**Stale mirror detection**: If `LastSync` for a placement is older than 2× `tickInterval`, the service attempts a synchronous self-heal `evaluate()` before failing the request.

### Dependency Injection (Uber fx)

All DI is runtime via `go.uber.org/fx`. The container is built in `cmd/server/main.go` using `fx.New(fx.Provide(...), fx.Invoke(...)).Run()`.

- `cmd/server/providers.go` — all provider functions and lifecycle registrations
- `cmd/server/main.go` — fx container wiring (interface bindings via `fx.Annotate`/`fx.As`, thin adapters for dual-type providers)

Workflow: add/modify a provider in `providers.go` → add it to `fx.Provide(...)` in `main.go`.

### Project Structure

```
cmd/server/
  main.go                  Entry point; boots Redis, Postgres, builds fx container and calls app.Run()
  providers.go             fx provider functions and lifecycle registrations
  handler/                 Gin handlers (routes, validator, handler_test)
  service/                 CMSDeliveryService — core evaluation + caching logic
  internal/evaluator/      LocalEvaluator — in-process rule evaluation

internal/
  delivery/http/           Router, middleware, DTOs, health check
  domain/entity/           GORM models + enums
  domain/repository/       Repository interfaces
  infrastructure/          cache (MemoryCache), database, logger, pubsub
  repository/              Concrete Postgres and Redis repository implementations
  http/client/             CLEN Lead and CustomerProfile HTTP clients (resty)

pkg/
  config/                  AppConfig via cleanenv (YAML + ENV override)
  ctxconsts/               Context key constants
  util/                    Shared helpers

configs/                   YAML config files (delivery.yaml is the entry point)
```

### Configuration

Config loads from `configs/delivery.yaml` first, then ENV vars override. See `.env.example` for all variables. Key vars: `PORT`, `DB_*`, `REDIS_*`, `CMS_RUNTIME_TTL`, `CMS_RUNTIME_INTERVAL`, `SWAGGER_HOST`, `SETENV` (`DEVLOCAL` for local dev, unset in AKS).

CLEN integrations (Lead, CustomerProfile) are configured separately via `CLEN_*` env vars and gracefully disabled when credentials are absent.

## Conventions

### Entity Design

All GORM models embed `entity.BaseModel` which provides:
- UUID primary key (`ID`)
- Audit fields (`CreatedAt`, `CreatedBy`, `UpdatedAt`, `UpdatedBy`)
- Soft-delete (`DeletedAt`)

### Formatting

`make fmt` runs `go fmt` **and** `scripts/format_tags.go` which aligns GORM/JSON struct tags. Always run before committing.

### Commit Messages

Enforced by `.githooks/commit-msg` (Conventional Commits): `type(scope): description`. Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`.

### Testing

- Tests co-located with source (`*_test.go` in the same package)
- Use `testify` for assertions, `go-sqlmock` for DB mocking, `redismock` for Redis mocking, `gofakeit` for test data

### External Integrations

- **CLEN Lead API** (`internal/http/client/clen_lead_client.go`) — fetches lead offerings for `SALES_TARGET` rules; nil client is tolerated so local dev boots without credentials
- **CLEN CustomerProfile API** — enriches user attributes; similarly nil-safe
- **Azure Key Vault / Managed Identity** — used in production (AKS); no Azure credentials needed for local dev
