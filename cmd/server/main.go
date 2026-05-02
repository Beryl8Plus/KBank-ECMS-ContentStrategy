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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	ecmsdocs "kbank-ecms/cmd/server/docs"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load .env file if present (ignored in production where env vars are injected)
	if loadErr := godotenv.Load(); loadErr != nil {
		logger.LStartup(ctx, entity.StartupLog{
			Service: "MAIN",
			Level:   "WARN",
			Message: "No .env file found, relying on environment variables",
		})
	}

	// Setup logger
	logger.LStartup(ctx, entity.StartupLog{Service: "CMS-DELIVERY", Level: "INFO", Message: "Starting cms-delivery pod"})

	// Load config — YAML provides non-secret defaults; ENV overrides credentials.
	cfg, err := config.LoadConfig("configs/delivery.yaml")
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "FATAL",
			Message: "Failed to load config: " + err.Error(),
		})
		os.Exit(1)
	}

	// Override swagger host from config (populated from SWAGGER_HOST env var).
	if cfg.Swagger.Host != "" {
		ecmsdocs.SwaggerInfo.Host = cfg.Swagger.Host
	}

	// Redis only — delivery service reads from cache, no PostgreSQL needed.
	redisCache, err := repository.NewRedisRepository(ctx, cfg.Redis)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "FATAL", Message: "Redis init failed: " + err.Error()})
		os.Exit(1)
	}

	// Initialize Postgres DB
	db, err := database.NewPostgresDB(cfg.Postgres)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "FATAL",
			Message: "Failed to initialize Postgres: " + err.Error(),
		})
		os.Exit(1)
	}

	// Build app — wires service → handler → middleware → router
	app, cleanup := InitializeApp(cfg, db, redisCache)
	defer cleanup()

	// Start background ticker (no-op if tickInterval <= 0).
	if err := app.Service.Start(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "FATAL", Message: "svc.Start failed: " + err.Error()})
		os.Exit(1)
	}

	port := cfg.Server.Port

	httpSrv := &http.Server{Addr: ":" + port, Handler: app.Router}

	// Run HTTP server in a background goroutine so main can wait on signal.
	go func() {
		logger.LStartup(ctx, entity.StartupLog{Service: "CMS-DELIVERY", Level: "INFO", Message: "Listening on :" + port})
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LSystem(context.Background(), entity.SystemLog{Service: "CMS-DELIVERY", Level: "FATAL", Message: "Server error: " + err.Error()})
			os.Exit(1)
		}
	}()

	// Block until Ctrl+C / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "INFO", Message: "Shutdown signal received"})

	// Stop the background ticker first, then drain the HTTP server.
	_ = app.Service.Stop()

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "WARN", Message: "HTTP shutdown error: " + err.Error()})
	}
	logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "INFO", Message: "Stopped"})
}
