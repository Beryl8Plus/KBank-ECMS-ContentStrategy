# KBank ECMS

This repository exposes the Rule Management API as a Go backend service structured following [golang-standards/project-layout](https://github.com/golang-standards/project-layout) with Clean Architecture principles.

- `POST /rule-management`: active API endpoint.

## Core Architectural Layers

The project is organized into four clean layers with a strict inward dependency rule — outer layers depend on inner layers, never the reverse.

```
┌──────────────────────────────────────────────────────────┐
│                    cmd/server/main.go                    │  Entry Point & Wiring
└─────────────────────────────┬────────────────────────────┘
                              │ depends on
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌─────────────────┐  ┌─────────────────┐  ┌────────────────────┐
│    delivery/    │  │    service/     │  │   repository/      │
│   http layer    │  │ business logic  │  │  data access impl  │
│                 │  │                 │  │                    │
│ - handler/      │  │ RuleManagement  │  │ RedisRepository    │
│ - middleware/   │  │   Service       │  │ AzureStorageRepo   │
│ - router.go     │  │                 │  │                    │
└────────┬────────┘  └───────┬─────────┘  └────────┬───────────┘
         │                   │                      │
         └───────────────────┼──────────────────────┘
                             │ all depend on
                             ▼
          ┌──────────────────────────────────────┐
          │            internal/domain/          │  Core (zero deps)
          │                                      │
          │  entity/          repository/        │
          │  - config.go      - cache.go         │
          │  - log.go         - storage.go       │
          └──────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Path | Responsibility |
|-------|------|----------------|
| **Domain** | `internal/domain/` | Entities, repository interfaces. No external dependencies. |
| **Use Case** | `internal/usecase/` | Business logic orchestration. Depends on domain only. |
| **Repository** | `internal/repository/` | Redis & Azure implementations. Implements domain interfaces. |
| **Delivery** | `internal/delivery/http/` | Gin HTTP handlers, middleware, route definitions. |
| **Infrastructure** | `internal/infrastructure/logger/` | Structured logging (cross-cutting concern). |
| **Pkg** | `pkg/util/` | Generic public utilities safe for external use. |

### Project Structure

```
├── cmd/
│   └── server/
│       └── main.go                          # Entry point — wires all layers
├── internal/
│   ├── domain/                              # Layer 1: Core entities & interfaces
│   │   ├── entity/
│   │   │   ├── config.go                   # InboundConfig, Server, RateLimit, RedisConfig
│   │   │   └── log.go                      # ErrorLog, RequestLog, SystemLog, etc.
│   │   └── repository/
│   │       ├── cache.go                    # CacheRepository interface
│   │       └── storage.go                  # StorageRepository interface
│   ├── usecase/                             # Layer 2: Business logic
│   │   └── rule_management_usecase.go
│   ├── repository/                          # Layer 3: Data access implementations
│   │   ├── redis_repository.go             # Implements CacheRepository
│   │   └── azure_repository.go             # Implements StorageRepository
│   ├── delivery/                            # Layer 4: HTTP delivery
│   │   └── http/
│   │       ├── handler/
│   │       │   └── rule_management_handler.go
│   │       ├── middleware/
│   │       │   └── middleware.go           # Rate limiter, concurrency, logger
│   │       └── router.go                   # Route definitions
│   └── infrastructure/
│       └── logger/
│           └── logger.go                   # Structured logging
├── pkg/
│   └── util/
│       └── string_slice.go                 # Generic utilities
├── configs/                                 # YAML configuration files
├── docs/                                    # API docs & diagrams
└── dockerfile
```

## Local Run

Set environment values and run:

```bash
# PowerShell
$env:SETENV="DEVLOCAL"
$env:REDIS_HOST="localhost"
$env:REDIS_PORT="6379"
go run ./cmd/server/

# Unix/macOS
SETENV=DEVLOCAL REDIS_HOST=localhost REDIS_PORT=6379 go run ./cmd/server/
```

Service listens on `:8081`.

## API Route

```bash
curl -X POST "http://localhost:8081/rule-management" \
  -H "requestID: req-002" \
  -H "Content-Type: application/json" \
  -d '{}'
```

## Config Files

- `configs/newservice_inbound_config.yaml` — inbound rate limit & server settings
- `configs/newservice_outbound_config.yaml` — outbound service settings
- `configs/redis_config.yaml` — Redis connection config

## Build

```bash
# Local build
go build ./cmd/server/

# Production (Docker)
docker build -f dockerfile -t kbank-ecms .

# Cross-compile for Linux
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kbank-ecms ./cmd/server/
```
