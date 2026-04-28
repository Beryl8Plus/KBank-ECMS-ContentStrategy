// @title						KBank ECMS API
// @version					1.0
// @description				Backend API for KBank ECMS Rule Management.
// @host						localhost:8081
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

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/util"

	ecmsdocs "kbank-ecms/docs/swagger/svc-contstrat-backoffice"
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
	logger.LStartup(ctx, entity.StartupLog{Service: "MAIN", Level: "INFO", Message: "Start App"})
	logger.LStartup(ctx, entity.StartupLog{Service: "MAIN", Level: "INFO", Message: "Loading runtime settings for new service"})

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

	// Initialize Redis Repository. If Redis is unavailable the publisher
	// becomes a no-op and cache invalidation falls back to TTL.
	redisCache, err := repository.NewRedisRepository(ctx, entity.RedisConfig{
		Host:     os.Getenv("REDIS_HOST"),
		Port:     os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "MAIN",
			Level:   "ERROR",
			Message: "Failed to initialize Redis: " + err.Error(),
		})
	}

	// Initialize Postgres DB
	db, err := database.NewPostgresDB(POSTGRES)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "MAIN",
			Level:   "FATAL",
			Message: "Failed to initialize Postgres: " + err.Error(),
		})
		os.Exit(1)
	}

	// Build router + occurrence worker via Google Wire
	app, err := InitializeApp(db, redisCache, rateLimit)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "MAIN",
			Level:   "FATAL",
			Message: "Failed to initialize app via Wire: " + err.Error(),
		})
		os.Exit(1)
	}

	// Start the occurrence materialization background worker.
	// It runs until ctx is cancelled (on SIGINT / SIGTERM below).
	go app.OccurrenceWorker.Start(ctx)

	// Start the attribute sync + integrity checker background worker.
	go app.AttributeSyncWorker.Start(ctx)

	port := "8081"
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: app.Router,
	}

	go func() {
		logger.LStartup(ctx, entity.StartupLog{
			Service: "MAIN",
			Level:   "INFO",
			Message: "Starting server on port " + port,
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LStartup(ctx, entity.StartupLog{
				Service: "MAIN",
				Level:   "FATAL",
				Message: "Failed to start server: " + err.Error(),
			})
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.LStartup(ctx, entity.StartupLog{Service: "MAIN", Level: "INFO", Message: "Shutdown signal received"})
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.LStartup(ctx, entity.StartupLog{
			Service: "MAIN",
			Level:   "ERROR",
			Message: "Server forced to shutdown: " + err.Error(),
		})
	}
	logger.LStartup(ctx, entity.StartupLog{Service: "MAIN", Level: "INFO", Message: "Server stopped"})
}
