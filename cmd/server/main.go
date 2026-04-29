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

	ecmsdocs "kbank-ecms/cmd/server/docs"
	"kbank-ecms/pkg/config"

	"github.com/joho/godotenv"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
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

	// Override swagger host from environment (e.g. staging.example.com)
	if swaggerHost := os.Getenv("SWAGGER_HOST"); swaggerHost != "" {
		ecmsdocs.SwaggerInfo.Host = swaggerHost
	}

	// Setup logger
	logger.LStartup(ctx, entity.StartupLog{Service: "CMS-DELIVERY", Level: "INFO", Message: "Starting cms-delivery pod"})

	POSTGRES := config.PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  "disable",
	}
	if ssl := os.Getenv("DB_SSLMODE"); ssl != "" {
		POSTGRES.SSLMode = ssl
	}

	// Redis only — delivery service reads from cache, no PostgreSQL needed.
	redisCache, err := repository.NewRedisRepository(ctx, config.RedisConfig{
		Host:     os.Getenv("REDIS_HOST"),
		Port:     os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "FATAL", Message: "Redis init failed: " + err.Error()})
		os.Exit(1)
	}

	// Initialize Postgres DB
	db, err := database.NewPostgresDB(POSTGRES)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "FATAL",
			Message: "Failed to initialize Postgres: " + err.Error(),
		})
		os.Exit(1)
	}

	// Load app config (timeouts, rate limits, etc.) from environment or config service.
	cfg := config.NewAppConfig()

	// Build app — wires service → handler → middleware → router
	app, cleanup := InitializeApp(cfg, db, redisCache)
	defer cleanup()

	// Start background ticker (no-op if tickInterval <= 0).
	if err := app.Service.Start(ctx); err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "CMS-DELIVERY", Level: "FATAL", Message: "svc.Start failed: " + err.Error()})
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

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

// parseDurationEnv reads an env var as a time.Duration string. Falls back to def.
func parseDurationEnv(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}
