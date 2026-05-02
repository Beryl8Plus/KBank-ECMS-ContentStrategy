# Design: Migrate Dependency Injection from Google Wire to Uber fx

**Date:** 2026-05-01  
**Author:** nrtsa  
**Status:** Approved

---

## Overview

Replace Google Wire (compile-time, codegen-based DI) with `go.uber.org/fx` (runtime, reflection-based DI with lifecycle management) across `cmd/server`. The migration eliminates the `wire.go` + `wire_gen.go` codegen step, moves Redis and Postgres initialisation into the fx container, and replaces manual goroutine + signal handling with fx's structured lifecycle hooks.

---

## Goals

- Remove `make wire-gen` step and all Wire codegen artefacts
- Move Redis and Postgres initialisation inside the fx container
- Replace manual HTTP server goroutine + OS signal handling with `fx.Lifecycle` hooks
- Keep all existing provider logic intact — no behaviour changes
- Maintain interface bindings (repository interfaces remain satisfied by concrete types)

---

## Non-Goals

- Splitting providers into `fx.Module` groups (deferred — not warranted at current scale)
- Adding new tests as part of this migration
- Changing any business logic in services, handlers, or repositories

---

## Approach: Standard fx idiom (Approach 2)

All dependencies — including Redis and Postgres — are provided through the fx container. `main.go` shrinks to `fx.New(...).Run()`. Lifecycle hooks manage startup and shutdown in correct drain order.

---

## File Changes

| File | Action | Reason |
|------|--------|--------|
| `cmd/server/wire.go` | **Delete** | Wire build-injection entrypoint, not needed with fx |
| `cmd/server/wire_gen.go` | **Delete** | Auto-generated Wire output, not needed with fx |
| `cmd/server/providers.go` | **Modify** | Remove `wire` import; adapt `ProvideCacheMemory`; add `ProvideConfig`, `ProvidePostgresDB`, `ProvideRedisRepository`, `RegisterServiceLifecycle`, `RegisterHTTPServer`, `RegisterSwaggerHost` |
| `cmd/server/main.go` | **Modify** | Replace manual init + `InitializeApp` + goroutine/signal block with `fx.New(...).Run()` |
| `go.mod` / `go.sum` | **Modify** | Remove `github.com/google/wire`; add `go.uber.org/fx` |
| `Makefile` | **Modify** | Remove `wire-gen` target; remove `wire` from `make init` install step |

The `App` struct (`Router` + `Service`) is **removed** — fx provides `*gin.Engine` and `*CMSDeliveryService` as direct container values consumed by lifecycle invocations.

---

## Provider Adaptations

### New providers (move infra init from `main.go` into container)

```go
func ProvideConfig() (config.AppConfig, error) {
    return config.LoadConfig("configs/delivery.yaml")
}

func ProvidePostgresDB(lc fx.Lifecycle, cfg config.AppConfig) (*gorm.DB, error) {
    db, err := database.NewPostgresDB(cfg.Postgres)
    if err != nil { return nil, err }
    lc.Append(fx.Hook{OnStop: func(ctx context.Context) error {
        sqlDB, _ := db.DB(); return sqlDB.Close()
    }})
    return db, nil
}

func ProvideRedisRepository(lc fx.Lifecycle, cfg config.AppConfig) (*repository.RedisRepository, error) {
    repo, err := repository.NewRedisRepository(context.Background(), cfg.Redis)
    if err != nil { return nil, err }
    lc.Append(fx.Hook{OnStop: func(ctx context.Context) error {
        return repo.Close()
    }})
    return repo, nil
}
```

### Modified provider: `ProvideCacheMemory`

Current signature: `func ProvideCacheMemory() (*deliveryservice.MemoryCache, func())`  
New signature: `func ProvideCacheMemory(lc fx.Lifecycle) *deliveryservice.MemoryCache`

The `func()` cleanup return is replaced by an `fx.Lifecycle` `OnStop` hook that stops all four in-memory caches.

### New lifecycle invocations (`fx.Invoke`)

```go
// RegisterServiceLifecycle starts and stops the background ticker.
func RegisterServiceLifecycle(lc fx.Lifecycle, svc *deliveryservice.CMSDeliveryService)

// RegisterHTTPServer starts ListenAndServe in OnStart, calls srv.Shutdown in OnStop.
func RegisterHTTPServer(lc fx.Lifecycle, cfg config.AppConfig, router *gin.Engine)

// RegisterSwaggerHost sets ecmsdocs.SwaggerInfo.Host from cfg when non-empty.
func RegisterSwaggerHost(cfg config.AppConfig)
```

### Interface binding — `wire.Bind` → `fx.Annotate` + `fx.As`

Each `wire.Bind` becomes an `fx.Annotate` wrapper on the provider in `fx.Provide`:

```go
// Before (wire.Bind in ProviderSet):
wire.Bind(new(domainrepo.ScheduleOccurrenceRepository), new(*repository.ScheduleOccurrencePostgresRepository))

// After (in main.go fx.Provide):
fx.Annotate(
    repository.NewScheduleOccurrencePostgresRepository,
    fx.As(new(domainrepo.ScheduleOccurrenceRepository)),
)
```

Full binding list:

| Interface | Concrete |
|-----------|----------|
| `domainrepo.ScheduleOccurrenceRepository` | `*repository.ScheduleOccurrencePostgresRepository` |
| `domainrepo.DecisionRuleRepository` | `*repository.DecisionRulePostgresRepository` |
| `domainrepo.LeadRepository` | `*repository.CLENLeadRepository` |
| `domainrepo.CustomerProfileRepository` | `*repository.CLENCustomerProfileRepository` |
| `domainrepo.CLENSchemaRegistryRepository` | `*repository.CLENSchemaRegistryPostgresRepository` |
| `domainrepo.AttributeRepository` | `*repository.AttributePostgresRepository` |
| `domainrepo.RedisCacheRepository` | `*repository.RedisRepository` (same instance as above) |
| `deliveryservice.RuntimeEvaluator` | `*evaluator.LocalEvaluator` |
| `deliveryservice.DeliveryService` | `*deliveryservice.CMSDeliveryService` (thin adapter) |

For `CMSDeliveryService`, both the concrete type and the interface are needed by different consumers (`RegisterServiceLifecycle` needs `*CMSDeliveryService`, `ProvideRouter` needs `DeliveryService`). The solution: provide the concrete type normally, add a thin adapter for the interface:

```go
fx.Provide(ProvideCMSDeliveryService)  // provides *CMSDeliveryService
fx.Provide(func(s *deliveryservice.CMSDeliveryService) deliveryservice.DeliveryService { return s })
```

fx resolves both from the same singleton — `ProvideCMSDeliveryService` is called once.

For `RedisCacheRepository`: `ProvideRedisRepository` already provides `*repository.RedisRepository`. A thin adapter provides the interface from the same instance:

```go
fx.Provide(func(r *repository.RedisRepository) domainrepo.RedisCacheRepository { return r })
```

---

## `main.go` Structure

```go
func main() {
    _ = godotenv.Load() // stays pre-fx; silently ignored in prod

    fx.New(
        fx.Provide(
            ProvideConfig,
            ProvidePostgresDB,
            ProvideRedisRepository,

            // Repos (interface-bound)
            fx.Annotate(repository.NewScheduleOccurrencePostgresRepository,
                fx.As(new(domainrepo.ScheduleOccurrenceRepository))),
            fx.Annotate(repository.NewDecisionRulePostgresRepository,
                fx.As(new(domainrepo.DecisionRuleRepository))),
            fx.Annotate(repository.NewCLENLeadRepository,
                fx.As(new(domainrepo.LeadRepository))),
            fx.Annotate(repository.NewCLENCustomerProfileRepository,
                fx.As(new(domainrepo.CustomerProfileRepository))),
            fx.Annotate(repository.NewCLENSchemaRegistryPostgresRepository,
                fx.As(new(domainrepo.CLENSchemaRegistryRepository))),
            fx.Annotate(repository.NewAttributePostgresRepository,
                fx.As(new(domainrepo.AttributeRepository))),
            func(r *repository.RedisRepository) domainrepo.RedisCacheRepository { return r },

            // CLEN clients + configs
            ProvideCLENLeadConfig,
            ProvideCLENLeadClient,
            ProvideCLENCustomerProfileConfig,
            ProvideCLENCustomerProfileClient,
            ProvideCustomerProfileEnrichConfig,

            // Core
            ProvideCacheMemory,
            fx.Annotate(ProvideRuntimeEvaluator,
                fx.As(new(deliveryservice.RuntimeEvaluator))),
            ProvideCMSDeliveryService,
            func(s *deliveryservice.CMSDeliveryService) deliveryservice.DeliveryService { return s },
            ProvideRouter,
        ),
        fx.Invoke(
            RegisterSwaggerHost,
            RegisterServiceLifecycle,
            RegisterHTTPServer,
        ),
    ).Run()
}
```

---

## Lifecycle / Shutdown Order

fx calls `OnStop` hooks in reverse registration order. The effective drain sequence is:

1. HTTP server `Shutdown(ctx)` — stops accepting new requests
2. Service ticker `Stop()` — stops background cache refresh
3. MemoryCache `Stop()` × 4 — stops in-memory cache eviction goroutines
4. Redis `Close()` — closes Redis connection pool
5. Postgres `Close()` — closes DB connection pool

This matches the correct order: stop traffic before stopping dependencies.

---

## fx Logging

By default `fx.New` emits its own structured startup/shutdown logs. To keep the existing logger style quiet or redirect fx logs, pass `fx.WithLogger(fxevent.NopLogger)` (silence) or a custom `fxevent.Logger` adapter to `fx.New`.

---

## Makefile Changes

```makefile
# Remove:
wire-gen:
    cd cmd/server && go run -mod=mod github.com/google/wire/cmd/wire

# Remove from init target:
go install github.com/google/wire/cmd/wire@latest
```

---

## Dependencies

```
# Remove:
github.com/google/wire

# Add:
go.uber.org/fx
```

---

## Testing

No new tests are added as part of this migration. Existing tests (`go test ./...`) must pass after the migration. Run with race detector (`go test -race ./...`) to verify no new data races from lifecycle hook goroutines.
