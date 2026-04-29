package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"kbank-ecms/pkg/auth"
	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Context keys for client authentication.
const (
	CtxKeyClientID = "client_id"
	CtxKeyScopes   = "scopes"
)

// IsScopeBypassEnabled reports whether scope checks should be bypassed.
// Controlled via the BYPASS_SCOPE_CHECK environment variable.
// WARNING: Use only in test/development environments.
func IsScopeBypassEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("BYPASS_SCOPE_CHECK")))
	return v == "true" || v == "1" || v == "yes"
}

// JWTMiddleware verifies JWT tokens from the Authorization header.
//
// Tokens may be either:
//   - User tokens issued for end-users (sets user_id in context)
//   - Client tokens issued via OAuth2 Client Credentials Flow (sets client_id + scopes)
//
// The middleware tries to verify the token as a client token first (which carries
// scopes for fine-grained authorization), and falls back to a user token. This
// allows the same endpoints to support both authentication methods.
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

		// 3a. Try verifying as a client token first (server-to-server flow)
		if clientClaims, err := jwtService.VerifyClientToken(parts[1]); err == nil && clientClaims.ClientID != "" {
			c.Set(CtxKeyClientID, clientClaims.ClientID)
			c.Set(CtxKeyScopes, clientClaims.Scopes)
			newCtx := context.WithValue(c.Request.Context(), ctxconsts.ClientIDKey, clientClaims.ClientID)
			c.Request = c.Request.WithContext(newCtx)
			c.Next()
			return
		}

		// 3b. Fall back to user token verification
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

// RequireScope returns a middleware that ensures the authenticated client token
// holds at least one of the required scopes (OR semantics).
//
// Scopes are formatted as "<source>:<action>" (e.g. "decision_rule:CREATE").
// This middleware is intended for client-credentials tokens; user tokens
// (which carry user_id) are passed through and should be authorized via
// ProfilePermissionGuard instead.
//
// Bypass: when the BYPASS_SCOPE_CHECK environment variable is truthy
// (true / 1 / yes), scope validation is skipped. WARNING: do not enable
// this in production — it disables fine-grained API authorization.
func RequireScope(source string, actions ...string) gin.HandlerFunc {
	required := make([]string, 0, len(actions))
	for _, action := range actions {
		required = append(required, fmt.Sprintf("%s:%s", source, action))
	}

	return func(c *gin.Context) {
		// Test bypass — skip all scope checks (use only in test/dev environments)
		if IsScopeBypassEnabled() {
			c.Next()
			return
		}

		// If the request was authenticated via a user token (no scopes set),
		// let the request through; ProfilePermissionGuard handles user RBAC.
		raw, exists := c.Get(CtxKeyScopes)
		if !exists {
			c.Next()
			return
		}

		clientScopes, ok := raw.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid_scope_format"})
			c.Abort()
			return
		}

		// Check if any of the required scopes is present (OR semantics)
		for _, scope := range clientScopes {
			for _, want := range required {
				if scope == want {
					c.Next()
					return
				}
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":           "insufficient_scope",
			"required_scopes": required,
			"granted_scopes":  clientScopes,
		})
		c.Abort()
	}
}
