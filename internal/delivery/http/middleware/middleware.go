package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kbank-ecms/pkg/config"
)

const TIMEOUT = 30 * time.Second

// Apply registers all global middleware onto an existing Gin engine.
// Called by the router layer after the engine is created. DBMiddleware is
// skipped when db is nil so services without a Postgres dependency can
// reuse this initializer.
func Apply(r *gin.Engine, db *gorm.DB, cfg config.AppConfig) {
	r.Use(CORSMiddleware())
	r.Use(ResponseHeaderMiddleware())
	r.Use(RateLimiterMiddleware(cfg.Server.Config.RateLimit.RPS, cfg.Server.Config.RateLimit.Burst))
	r.Use(ConcurrencyMiddleware(cfg.Server.Config.RateLimit.MCR))
	r.Use(LoggerMiddleware())
	r.Use(TimeoutMiddleware(cfg.Server.Config.Timeout.ReqCtxTimeout))
	if db != nil {
		r.Use(DBMiddleware(db, cfg.Server.Config.Timeout.DBCtxTimeout))
	}
}
