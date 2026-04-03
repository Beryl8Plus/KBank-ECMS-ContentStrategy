package main

import (
	"context"
	"kbank-ecms/internal/middleware"
	"kbank-ecms/internal/model"
	newserviceapp "kbank-ecms/internal/newservice/app"
	"kbank-ecms/internal/util"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	ctx := context.Background()

	// Setup logger
	util.LStartup(model.StartupLog{Service: "MAIN", Level: "INFO", Message: "Start App"})
	util.LStartup(model.StartupLog{Service: "MAIN", Level: "INFO", Message: "Loading runtime settings for new service"})

	REDIS := model.RedisConfig{
		Host:     os.Getenv("REDIS_HOST"),
		Port:     os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}

	rateLimit := model.RateLimit{RPS: 50, Burst: 100, MCR: 10}
	if cfgRateLimit, err := loadNewServiceRateLimit("./configs/newservice_inbound_config.yaml"); err == nil {
		rateLimit = cfgRateLimit
	}

	// Initialize Redis
	if err := util.InitRedis(ctx, REDIS); err != nil {
		util.LSystem(model.SystemLog{
			Service: "MAIN",
			Level:   "ERROR",
			Message: "Failed to initialize Redis: " + err.Error(),
		})
		// Proceeding without Redis for now, or return depending on requirement
	}

	// Initialize Gin
	r := gin.Default()

	r.Use(middleware.RateLimiterMiddleware(rateLimit.RPS, rateLimit.Burst))
	r.Use(middleware.ConcurrencyMiddleware(rateLimit.MCR))
	r.Use(middleware.LoggerMiddleware())

	// Define Routes
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/rule-management", func(c *gin.Context) {
		newserviceapp.IngressRuleManagement(c)
	})

	// Start Server
	port := "8081" // Default port or from config
	util.LStartup(model.StartupLog{
		Service: "MAIN",
		Level:   "INFO",
		Message: "Starting server on port " + port,
	})
	if err := r.Run(":" + port); err != nil {
		util.LStartup(model.StartupLog{
			Service: "MAIN",
			Level:   "FATAL",
			Message: "Failed to start server: " + err.Error(),
		})
	}
}

func loadNewServiceRateLimit(path string) (model.RateLimit, error) {
	var cfg model.InboundConfig
	body, err := os.ReadFile(path)
	if err != nil {
		return model.RateLimit{}, err
	}

	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return model.RateLimit{}, err
	}

	if len(cfg.Server) == 0 {
		return model.RateLimit{}, os.ErrNotExist
	}

	return cfg.Server[0].RateLimit, nil
}
