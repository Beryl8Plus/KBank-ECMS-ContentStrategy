package middleware

import (
	"kbank-ecms/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Apply registers all global middleware onto an existing Gin engine.
// Called by the router layer after the engine is created.
func Apply(r *gin.Engine, db *gorm.DB, rateLimit entity.RateLimit) {
	r.Use(RateLimiterMiddleware(rateLimit.RPS, rateLimit.Burst))
	r.Use(ConcurrencyMiddleware(rateLimit.MCR))
	r.Use(LoggerMiddleware())
	r.Use(UserIDMiddleware())
	r.Use(DBMiddleware(db))
}
