package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockPermissionRepo is a test double for domainrepo.PermissionRepository.
type mockPermissionRepo struct {
	fn func(ctx context.Context, userID uuid.UUID, source, action string) (bool, error)
}

func (m *mockPermissionRepo) HasPermission(ctx context.Context, userID uuid.UUID, source, action string) (bool, error) {
	return m.fn(ctx, userID, source, action)
}

func setupGuardRouter(repo *mockPermissionRepo, source string, actions ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", ProfilePermissionGuard(repo, source, actions...), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestProfilePermissionGuard_NoUserIDInContext_Returns401(t *testing.T) {
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
		return true, nil
	}}
	r := setupGuardRouter(repo, "decision_rule", "create")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfilePermissionGuard_InvalidUserIDType_Returns401(t *testing.T) {
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
		return true, nil
	}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(string(ctxconsts.UserIDKey), "not-a-uuid-struct")
		c.Next()
	}, ProfilePermissionGuard(repo, "decision_rule", "create"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfilePermissionGuard_RepoError_Returns500(t *testing.T) {
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
		return false, errors.New("db error")
	}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(string(ctxconsts.UserIDKey), uuid.New())
		c.Next()
	}, ProfilePermissionGuard(repo, "decision_rule", "create"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProfilePermissionGuard_NotAllowed_Returns403(t *testing.T) {
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
		return false, nil
	}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(string(ctxconsts.UserIDKey), uuid.New())
		c.Next()
	}, ProfilePermissionGuard(repo, "decision_rule", "create"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestProfilePermissionGuard_Allowed_PassesThrough(t *testing.T) {
	calledWith := struct {
		source string
		action string
	}{}
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, source, action string) (bool, error) {
		calledWith.source = source
		calledWith.action = action
		return true, nil
	}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(string(ctxconsts.UserIDKey), uuid.New())
		c.Next()
	}, ProfilePermissionGuard(repo, "decision_rule", "view_all"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "decision_rule", calledWith.source)
	assert.Equal(t, "view_all", calledWith.action)
}

func TestProfilePermissionGuard_MultipleActions_GrantedOnAny(t *testing.T) {
	// Only "delete" returns true; create and edit return false.
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, _, action string) (bool, error) {
		return action == "delete", nil
	}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(string(ctxconsts.UserIDKey), uuid.New())
		c.Next()
	}, ProfilePermissionGuard(repo, "decision_rule", "create", "edit", "delete"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilePermissionGuard_MultipleActions_DeniedWhenNoneMatch(t *testing.T) {
	// All actions denied.
	repo := &mockPermissionRepo{fn: func(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
		return false, nil
	}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set(string(ctxconsts.UserIDKey), uuid.New())
		c.Next()
	}, ProfilePermissionGuard(repo, "decision_rule", "create", "edit", "delete"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
