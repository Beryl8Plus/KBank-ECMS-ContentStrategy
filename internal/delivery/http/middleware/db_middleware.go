package middleware

import (
	"context"
	"fmt"
	"time"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DBMiddleware integrates GORM's DB instance into the request context.
// Similar to the pattern described in: https://gorm.io/docs/context.html#Integration-with-Chi-Middleware
// but adapted for the Gin framework. It also sets a default timeout for all database operations within the request scope.
func DBMiddleware(db *gorm.DB, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		timeoutContext, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer func() {
			if err := timeoutContext.Err(); err != nil {
				logger.LSystem(timeoutContext, entity.SystemLog{
					Service:       "DB-MIDDLEWARE",
					Level:         "ERROR",
					CorrelationID: c.GetHeader("requestID"),
					Message:       fmt.Sprintf("DB context cancelled or timed out: %v", err),
				})
			}
			cancel()
		}()

		// Set the derived timeout context over the DB instance
		dbWithCtx := db.WithContext(timeoutContext)

		// Set in Gin context (optional, depending on if handlers prefer gin.Context wrapper)
		c.Set(string(ctxconsts.DBKey), dbWithCtx)

		// Set in standard request context (essential for GORM hooks or if using c.Request.Context() down the line)
		newCtx := context.WithValue(c.Request.Context(), ctxconsts.DBKey, dbWithCtx)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}
