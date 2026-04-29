package middleware

import (
	"context"
	"net/http"
	"strings"

	"kbank-ecms/pkg/auth"
	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// JWTMiddleware verifies JWT tokens from Authorization header
// and sets user information in the Gin context for downstream handlers.
// Following Gin Framework authentication standards.
func JWTMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract Token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 2. Validate Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// 3. Verify JWT token
		claims, err := jwtService.VerifyToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 4. Store user information in Gin context
		c.Set("currentUser", claims)
		c.Set(ctxconsts.UserIDKey, claims.UserID)

		// 5. Append to request context for GORM hooks and downstream reads
		newCtx := context.WithValue(c.Request.Context(), ctxconsts.UserIDKey, claims.UserID)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}

// UserIDMiddleware extracts the user ID from the "X-User-Id" header
// and sets it in both the Gin context and the standard request context.
// This is a fallback middleware for backward compatibility.
// TODO: Deprecate this in favor of JWTMiddleware once migration is complete.
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIdStr := c.GetHeader("X-User-Id")
		if userIdStr != "" {
			uid, err := uuid.Parse(userIdStr)
			if err == nil {
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

// ClientAuthMiddleware validates client JWT tokens for server-to-server communication.
// This middleware is specifically for OAuth2 Client Credentials Flow.
func ClientAuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract Token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 2. Validate Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// 3. Verify JWT token as client token
		claims, err := jwtService.VerifyClientToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 4. Store client information in Gin context
		c.Set("client_id", claims.ClientID)
		c.Set("scopes", claims.Scopes)

		// 5. Append to request context for downstream reads
		newCtx := context.WithValue(c.Request.Context(), "client_id", claims.ClientID)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}

// RequireScope middleware checks if the client token has the required scope.
// This should be used after ClientAuthMiddleware.
func RequireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopes, exists := c.Get("scopes")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "scope information not found"})
			c.Abort()
			return
		}

		clientScopes, ok := scopes.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid scope format"})
			c.Abort()
			return
		}

		// Check if required scope is present
		hasScope := false
		for _, scope := range clientScopes {
			if scope == requiredScope {
				hasScope = true
				break
			}
		}

		if !hasScope {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient_scope"})
			c.Abort()
			return
		}

		c.Next()
	}
}

