package router

import (
	"kbank-ecms/internal/delivery/http/middleware"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/pkg/healthcheck"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// InitNewRouter creates the Gin engine and wires all layers in order:
// service → handler → middleware → router
func InitNewRouter(db *gorm.DB, rateLimit entity.RateLimit, redisClient *redis.Client) *gin.Engine {
	r := gin.New()

	// Middleware layer — applied globally before any handler
	middleware.Apply(r, db, rateLimit)

	// System routes (observability, docs) — no auth/permission guards
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint at /healthz
	healthcheck.Register(r, db, redisClient)

	return r
}
