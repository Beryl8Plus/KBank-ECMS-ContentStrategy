package middleware

import (
	"context"

	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserIDMiddleware extracts the user ID from the "X-User-Id" header
// and sets it in both the Gin context and the standard request context.
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIdStr := c.GetHeader("X-User-Id")
		if userIdStr != "" {
			uid, err := uuid.Parse(userIdStr)
			if err == nil {
				c.Set(ctxconsts.UserIDKey, uid)
				newCtx := context.WithValue(c.Request.Context(), ctxconsts.UserIDKey, uid)
				c.Request = c.Request.WithContext(newCtx)
			}
		}

		c.Next()
	}
}