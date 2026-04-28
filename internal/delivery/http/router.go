package router

import (
	"kbank-ecms/internal/delivery/http/middleware"
	"kbank-ecms/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// InitNewRouter creates the Gin engine and wires all layers in order:
// service → handler → middleware → router
func InitNewRouter(db *gorm.DB, rateLimit entity.RateLimit) *gin.Engine {
	r := gin.New()

	// Middleware layer — applied globally before any handler
	middleware.Apply(r, db, rateLimit)

	// System routes (observability, docs) — no auth/permission guards
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
