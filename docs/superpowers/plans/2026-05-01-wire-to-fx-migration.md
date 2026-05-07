# Wire-to-fx Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Google Wire compile-time DI with `go.uber.org/fx` runtime DI, moving Redis and Postgres initialisation into the container and replacing manual goroutine + signal handling with fx lifecycle hooks.

**Architecture:** All dependencies — including Redis and Postgres — are provided through an `fx.Container`. Provider functions live in `cmd/server/providers.go`; `cmd/server/main.go` becomes `fx.New(...).Run()`. Lifecycle hooks (OnStart/OnStop) replace the current cleanup func pattern and manual OS-signal goroutine.

**Tech Stack:** `go.uber.org/fx v1.x`, `go.uber.org/fx/fxevent` (NopLogger), existing GORM / go-redis / Gin stack unchanged.

**Spec:** `docs/superpowers/specs/2026-05-01-wire-to-fx-migration-design.md`

---

## File Map

| File | Change |
|------|--------|
| `cmd/server/wire.go` | Delete |
| `cmd/server/wire_gen.go` | Delete |
| `cmd/server/providers.go` | Modify — add imports, 6 new functions, change `ProvideCacheMemory`, remove `App` struct + `ProvideApp` + `ProviderSet` |
| `cmd/server/main.go` | Rewrite — `fx.New(...).Run()` replaces manual init + `InitializeApp` + goroutine/signal block |
| `Makefile` | Modify — remove `wire-gen` target + wire from `init` |
| `go.mod` / `go.sum` | Modify — `go mod tidy` removes wire, adds fx |

---

## Task 1: Add `go.uber.org/fx` dependency and record baseline

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add fx**

```bash
cd /path/to/repo
go get go.uber.org/fx@latest
```

Expected: `go.mod` now contains `go.uber.org/fx v1.x.x` in `require`.

- [ ] **Step 2: Verify existing build still passes (wire still intact)**

```bash
go build ./...
```

Expected: exits 0, no errors.

- [ ] **Step 3: Record baseline test results**

```bash
go test ./...
```

Expected: all tests pass (note any pre-existing skips/failures so you don't confuse them with regressions later).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add go.uber.org/fx"
```

---

## Task 2: Add new lifecycle providers to `providers.go` (additive — no breaking changes)

This task only *adds* new functions and imports to `providers.go`. Nothing existing is touched. The file still compiles with wire present.

**Files:**
- Modify: `cmd/server/providers.go`

- [ ] **Step 1: Update the import block**

Replace the existing `import (...)` block in `cmd/server/providers.go` with:

```go
import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"

	ecmsdocs "kbank-ecms/cmd/server/docs"
	cmshandler "kbank-ecms/cmd/server/handler"
	evaluator "kbank-ecms/cmd/server/internal/evaluator"
	deliveryservice "kbank-ecms/cmd/server/service"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/config"
)
```

- [ ] **Step 2: Append `ProvideConfig` at the bottom of `providers.go`**

```go
// ProvideConfig loads AppConfig so fx can inject it throughout the container.
func ProvideConfig() (config.AppConfig, error) {
	return config.LoadConfig("configs/delivery.yaml")
}
```

- [ ] **Step 3: Append `ProvidePostgresDB`**

```go
// ProvidePostgresDB opens the Postgres connection and registers an OnStop hook to close it.
func ProvidePostgresDB(lc fx.Lifecycle, cfg config.AppConfig) (*gorm.DB, error) {
	db, err := database.NewPostgresDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			sqlDB, _ := db.DB()
			return sqlDB.Close()
		},
	})
	return db, nil
}
```

- [ ] **Step 4: Append `ProvideRedisRepository`**

```go
// ProvideRedisRepository connects to Redis and registers an OnStop hook to close the client.
func ProvideRedisRepository(lc fx.Lifecycle, cfg config.AppConfig) (*repository.RedisRepository, error) {
	repo, err := repository.NewRedisRepository(context.Background(), cfg.Redis)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return repo.Client().Close()
		},
	})
	return repo, nil
}
```

- [ ] **Step 5: Append `RegisterServiceLifecycle`**

```go
// RegisterServiceLifecycle starts the background evaluation ticker on app start
// and stops it on shutdown.
func RegisterServiceLifecycle(lc fx.Lifecycle, svc *deliveryservice.CMSDeliveryService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return svc.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return svc.Stop()
		},
	})
}
```

- [ ] **Step 6: Append `RegisterHTTPServer`**

```go
// RegisterHTTPServer starts the Gin HTTP server in a goroutine on app start
// and shuts it down gracefully on stop.
func RegisterHTTPServer(lc fx.Lifecycle, cfg config.AppConfig, router *gin.Engine) {
	srv := &http.Server{Addr: ":" + cfg.Server.Port, Handler: router}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "CMS-DELIVERY", Level: "INFO",
					Message: "Listening on :" + cfg.Server.Port,
				})
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.LSystem(context.Background(), entity.SystemLog{
						Service: "CMS-DELIVERY", Level: "FATAL",
						Message: "Server error: " + err.Error(),
					})
					os.Exit(1)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
```

- [ ] **Step 7: Append `RegisterSwaggerHost`**

```go
// RegisterSwaggerHost overrides the Swagger UI host when SWAGGER_HOST env var is set.
func RegisterSwaggerHost(cfg config.AppConfig) {
	if cfg.Swagger.Host != "" {
		ecmsdocs.SwaggerInfo.Host = cfg.Swagger.Host
	}
}
```

- [ ] **Step 8: Verify build still passes (wire intact, new functions are additive)**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 9: Commit**

```bash
git add cmd/server/providers.go
git commit -m "feat(di): add fx lifecycle providers to providers.go"
```

---

## Task 3: Atomic swap — delete Wire files, adapt `providers.go`, rewrite `main.go`

This task makes the breaking changes. Steps 1–4 must all be completed before the build is clean again. Do **not** commit mid-way.

**Files:**
- Delete: `cmd/server/wire.go`
- Delete: `cmd/server/wire_gen.go`
- Modify: `cmd/server/providers.go`
- Rewrite: `cmd/server/main.go`

- [ ] **Step 1: Delete `wire.go`**

```bash
rm cmd/server/wire.go
```

This file has `//go:build wireinject` so it was excluded from normal builds — deletion has no immediate compile impact.

- [ ] **Step 2: Delete `wire_gen.go`**

```bash
rm cmd/server/wire_gen.go
```

`wire_gen.go` defined `InitializeApp` which `main.go` still calls — the build is now broken until Step 4.

- [ ] **Step 3: Adapt `cmd/server/providers.go`**

**3a.** Replace the import block — remove `"github.com/google/wire"`:

```go
import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"

	ecmsdocs "kbank-ecms/cmd/server/docs"
	cmshandler "kbank-ecms/cmd/server/handler"
	evaluator "kbank-ecms/cmd/server/internal/evaluator"
	deliveryservice "kbank-ecms/cmd/server/service"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/config"
)
```

**3b.** Replace `ProvideCacheMemory` — change signature to accept `fx.Lifecycle`, remove the `func()` cleanup return:

```go
// ProvideCacheMemory builds the L1 in-process cache and registers OnStop hooks
// to stop each cache's eviction goroutine.
func ProvideCacheMemory(lc fx.Lifecycle) *deliveryservice.MemoryCache {
	schedules := cache.NewCacheMemory[[]*entity.Schedule]("cms-runtime", 0.60, 24*time.Hour)
	decisionRule := cache.NewCacheMemory[*entity.DecisionRule]("cms-runtime", 0.60, 24*time.Hour)
	versionHashes := cache.NewCacheMemory[string]("cms-runtime-versions", 0.60, 24*time.Hour)
	lastSync := cache.NewCacheMemory[time.Time]("cms-runtime-syncs", 0.60, 24*time.Hour)
	memoryCache := deliveryservice.MemoryCache{
		Schedules:     schedules,
		DecisionRule:  decisionRule,
		VersionHashes: versionHashes,
		LastSync:      lastSync,
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			schedules.Stop()
			decisionRule.Stop()
			versionHashes.Stop()
			lastSync.Stop()
			return nil
		},
	})
	return &memoryCache
}
```

**3c.** Delete the `App` struct and `ProvideApp` function. Remove this entire block:

```go
// App groups the dependencies needed by main.
type App struct {
	Router  *gin.Engine
	Service *deliveryservice.CMSDeliveryService
}

// ProvideApp creates the App struct.
func ProvideApp(r *gin.Engine, svc *deliveryservice.CMSDeliveryService) *App {
	return &App{
		Router:  r,
		Service: svc,
	}
}
```

**3d.** Delete the `ProviderSet` variable. Remove this entire block:

```go
// ProviderSet definition
var ProviderSet = wire.NewSet(
	...
)
```

(Remove from `// ProviderSet definition` through the closing `)` of `wire.NewSet`.)

- [ ] **Step 4: Rewrite `cmd/server/main.go`**

Replace the entire file content with:

```go
// @title						KBank ECMS CMS Delivery API
// @version					1.0
// @description				Backend API for KBank ECMS CMS Delivery Runtime Service.
// @host						localhost:8082
// @BasePath					/
// @securityDefinitions.apikey	XUserIdAuth
// @in							header
// @name						X-User-Id
package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/fx"

	deliveryservice "kbank-ecms/cmd/server/service"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
)

func main() {
	_ = godotenv.Load()
	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "CMS-DELIVERY", Level: "INFO", Message: "Starting cms-delivery pod",
	})

	app := fx.New(
		fx.Provide(
			ProvideConfig,
			ProvidePostgresDB,
			ProvideRedisRepository,

			// Repository interface bindings
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

			// RedisCacheRepository satisfied from the same *RedisRepository singleton
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
			// DeliveryService interface satisfied from the same *CMSDeliveryService singleton
			func(s *deliveryservice.CMSDeliveryService) deliveryservice.DeliveryService { return s },
			ProvideRouter,
		),
		fx.Invoke(
			RegisterSwaggerHost,
			RegisterServiceLifecycle,
			RegisterHTTPServer,
		),
	)

	if err := app.Err(); err != nil {
		logger.LSystem(context.Background(), entity.SystemLog{
			Service: "CMS-DELIVERY", Level: "FATAL",
			Message: "Container failed to initialise: " + err.Error(),
		})
		os.Exit(1)
	}

	app.Run()
}
```

- [ ] **Step 5: Verify the build is clean**

```bash
go build ./...
```

Expected: exits 0, no errors.

- [ ] **Step 6: Run existing tests**

```bash
go test ./...
```

Expected: same results as baseline recorded in Task 1.

- [ ] **Step 7: Run with race detector**

```bash
go test -race ./...
```

Expected: no new data races.

- [ ] **Step 8: Commit**

```bash
git add cmd/server/providers.go cmd/server/main.go
git commit -m "refactor(di): replace Google Wire with uber-go/fx"
```

---

## Task 4: Update `Makefile`

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Remove `wire-gen` from the `.PHONY` line**

Change line 1 from:

```makefile
.PHONY: init build run dev-build dev-up dev-down test vet lint fmt format-tags clean install-hooks swagger swagger-format swagger-server wire-gen
```

To:

```makefile
.PHONY: init build run dev-build dev-up dev-down test vet lint fmt format-tags clean install-hooks swagger swagger-format swagger-server
```

- [ ] **Step 2: Remove the `wire-gen` target**

Delete these 3 lines from the Makefile:

```makefile
## Generate wire dependencies
wire-gen:
	wire gen ./cmd/server
```

- [ ] **Step 3: Remove wire from the `init` target**

Delete these 2 lines from the `init` target:

```makefile
	@echo "Installing wire..."
	go install github.com/google/wire/cmd/wire@latest
```

- [ ] **Step 4: Verify `make init` output no longer mentions wire**

```bash
grep -n "wire" Makefile
```

Expected: no matches (or only matches in comments unrelated to the install/target, if any).

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "chore(makefile): remove wire-gen target and wire install"
```

---

## Task 5: Remove wire from go.mod and final verification

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Run `go mod tidy`**

```bash
go mod tidy
```

Expected: `github.com/google/wire` removed from `go.mod` and `go.sum`. `go.uber.org/fx` and its transitive deps appear in `go.sum`.

- [ ] **Step 2: Verify wire is gone from go.mod**

```bash
grep "google/wire" go.mod
```

Expected: no output.

- [ ] **Step 3: Final build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 4: Final test run with race detector**

```bash
go test -race ./...
```

Expected: all tests pass, no races.

- [ ] **Step 5: Run `make fmt` to clean up any import ordering**

```bash
make fmt
```

- [ ] **Step 6: Stage any fmt changes and commit**

```bash
git add go.mod go.sum
git add -p  # review any fmt changes
git commit -m "chore(deps): remove google/wire, run go mod tidy"
```

---

## Shutdown Order Reference

fx calls OnStop hooks in reverse order of registration. With the providers registered as above, shutdown drains in this sequence:

1. **HTTP server** `Shutdown(ctx)` — stops accepting new requests
2. **Service ticker** `Stop()` — stops background cache refresh goroutine
3. **MemoryCache** × 4 `Stop()` — stops in-memory eviction goroutines
4. **Redis** `Client().Close()` — closes connection pool
5. **Postgres** `sqlDB.Close()` — closes connection pool

This is the correct drain order (traffic stops before infrastructure closes).
