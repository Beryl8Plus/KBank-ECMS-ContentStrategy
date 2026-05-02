package router

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"kbank-ecms/internal/delivery/http/healthcheck"
	"kbank-ecms/internal/delivery/http/middleware"
	"kbank-ecms/pkg/config"
)

// InitNewRouter creates the Gin engine and wires all layers in order:
// service → handler → middleware → router
func InitNewRouter(cfg config.AppConfig, db *gorm.DB, redisClient *redis.Client) *gin.Engine {
	r := gin.New()

	// Middleware layer — applied globally before any handler
	middleware.Apply(r, db, cfg)

	// System routes (observability, docs) — no auth/permission guards
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint at /healthz
	healthcheck.Register(r, db, redisClient)

	return r
}
