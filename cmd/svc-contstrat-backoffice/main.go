// @title KBank ECMS API
// @version 1.0
// @description Backend API for KBank ECMS Rule Management.
// @host localhost:8081
// @BasePath /
// @securityDefinitions.apikey XUserIdAuth
// @in header
// @name X-User-Id
package main

import (
	"context"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/util"
	"os"

	ecmsdocs "kbank-ecms/docs/swagger/svc-contstrat-backoffice"

	"github.com/joho/godotenv"
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

	// Setup logger
	logger.LStartup(ctx, entity.StartupLog{Service: "MAIN", Level: "INFO", Message: "Start App"})
	logger.LStartup(ctx, entity.StartupLog{Service: "MAIN", Level: "INFO", Message: "Loading runtime settings for new service"})

	REDIS := entity.RedisConfig{
		Host:     os.Getenv("REDIS_HOST"),
		Port:     os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}

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

	// Initialize Redis Repository
	if _, err := repository.NewRedisRepository(ctx, REDIS); err != nil {
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

	// Build router — wires service → handler → middleware → router via Google Wire
	r, err := InitializeApp(db, rateLimit)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "MAIN",
			Level:   "FATAL",
			Message: "Failed to initialize app via Wire: " + err.Error(),
		})
		os.Exit(1)
	}

	// Start Server
	port := "8081" // Default port or from config
	logger.LStartup(ctx, entity.StartupLog{
		Service: "MAIN",
		Level:   "INFO",
		Message: "Starting server on port " + port,
	})
	if err := r.Run(":" + port); err != nil {
		logger.LStartup(ctx, entity.StartupLog{
			Service: "MAIN",
			Level:   "FATAL",
			Message: "Failed to start server: " + err.Error(),
		})
	}
}
