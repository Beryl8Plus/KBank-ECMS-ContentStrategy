package middleware

import (
	"context"

	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserIDMiddleware extracts the user ID from the "X-User-Id" header
// and sets it in both the Gin context and the standard request context.
// This is required for downstream components like GORM auto-stamping and profile permission guards.
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIdStr := c.GetHeader("X-User-Id") // TODO: get user id from token instead of x-user-id header
		if userIdStr != "" {
			if uid, err := uuid.Parse(userIdStr); err == nil {
				// Set in Gin context for profile permission guard
				c.Set(ctxconsts.UserIDKey, uid)
				// Append to request context for GORM hooks and downstream c.Request.Context() reads
				newCtx := context.WithValue(c.Request.Context(), ctxconsts.UserIDKey, uid)
				c.Request = c.Request.WithContext(newCtx)
			}
		}

		c.Next()
	}
}
