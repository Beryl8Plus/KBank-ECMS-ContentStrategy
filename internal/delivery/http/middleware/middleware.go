package middleware

import (
	"kbank-ecms/pkg/config"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const TIMEOUT = 30 * time.Second

// Apply registers all global middleware onto an existing Gin engine.
// Called by the router layer after the engine is created.
func Apply(r *gin.Engine, db *gorm.DB, cfg config.AppConfig) {
	r.Use(CORSMiddleware())
	r.Use(ResponseHeaderMiddleware())
	r.Use(RateLimiterMiddleware(cfg.RateLimit.RPS, cfg.RateLimit.Burst))
	r.Use(ConcurrencyMiddleware(cfg.RateLimit.MCR))
	r.Use(LoggerMiddleware())
	r.Use(UserIDMiddleware())
	r.Use(TimeoutMiddleware(cfg.Timeout.ReqCtxTimeout))
	r.Use(DBMiddleware(db, cfg.Timeout.DBCtxTimeout))
}
