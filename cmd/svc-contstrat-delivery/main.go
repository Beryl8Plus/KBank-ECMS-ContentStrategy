// @title KBank ECMS CMS Delivery API
// @version 1.0
// @description Backend API for KBank ECMS CMS Delivery Service.
// @host localhost:8082
// @BasePath /
// @securityDefinitions.apikey XUserIdAuth
// @in header
// @name X-User-Id
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ecmsdocs "kbank-ecms/docs/swagger/svc-contstrat-delivery"

	"github.com/joho/godotenv"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/util"
)

func main() {

	ctx := context.Background()

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

	logger.LStartup(ctx, entity.StartupLog{Service: "CMS-DELIVERY", Level: "INFO", Message: "Starting cms-delivery pod"})

	rateLimit := entity.RateLimit{RPS: 50, Burst: 100, MCR: 10}
	if cfgRateLimit, err := util.LoadNewServiceRateLimit("./configs/newservice_inbound_config.yaml"); err == nil {
		rateLimit = cfgRateLimit
	}

	POSTGRES := entity.PostgresConfig{
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
	redisCache, err := repository.NewRedisRepository(ctx, entity.RedisConfig{
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

	// Build app — wires service → handler → middleware → router
	app, cleanup := InitializeApp(db, redisCache, rateLimit)
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
