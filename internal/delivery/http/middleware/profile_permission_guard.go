package middleware

import (
	"net/http"

	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProfilePermissionGuard returns a middleware that verifies the authenticated user's
// profile holds at least one of the required permissions (OR semantics).
//
// Permission matrix (Content Decision Rule):
//
//	User Group                  | Create | Edit | Del | View All | Edit All | Del All
//	Content Strategy Marker     |  true  | true | true|   true   |  false   |  false
//	Content Strategy Super Admin|  true  | true | true|   true   |   true   |   true
//	IT Admin                    | false  |false |false|   true   |   true   |   true
//	Viewer                      | false  |false |false|   true   |  false   |  false
//
// Edit/Delete apply to own records only; Edit All/Delete All apply to all records.
//
// It expects the user ID (uuid.UUID) to be stored in Gin context under ctxconsts.UserIDKey,
// set by an upstream authentication middleware.
//
// Example — require any of create, edit, or delete:
//
//	ProfilePermissionGuard(repo, enums.PermissionSourceDecisionRule,
//	    enums.PermissionActionCreate, enums.PermissionActionEdit, enums.PermissionActionDelete)
func ProfilePermissionGuard(repo domainrepo.PermissionRepository, source string, actions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawID, exists := c.Get(string(ctxconsts.UserIDKey))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID, ok := rawID.(uuid.UUID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		for _, action := range actions {
			allowed, err := repo.HasPermission(c.Request.Context(), userID, source, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
			if allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}
