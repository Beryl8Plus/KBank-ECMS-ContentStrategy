package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Valid UUID Header", func(t *testing.T) {
		uid := uuid.New()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-User-Id", uid.String())
		w := httptest.NewRecorder()

		r := gin.New()
		r.Use(UserIDMiddleware())
		r.GET("/", func(c *gin.Context) {
			// Check Gin context
			ginUid, exists := c.Get(ctxconsts.UserIDKey)
			assert.True(t, exists)
			assert.Equal(t, uid, ginUid)

			// Check request context
			reqUid := c.Request.Context().Value(ctxconsts.UserIDKey)
			assert.NotNil(t, reqUid)
			assert.Equal(t, uid, reqUid)

			c.String(http.StatusOK, "OK")
		})

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid UUID Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-User-Id", "invalid-uuid")
		w := httptest.NewRecorder()

		r := gin.New()
		r.Use(UserIDMiddleware())
		r.GET("/", func(c *gin.Context) {
			_, exists := c.Get(ctxconsts.UserIDKey)
			assert.False(t, exists)

			reqUid := c.Request.Context().Value(ctxconsts.UserIDKey)
			assert.Nil(t, reqUid)

			c.String(http.StatusOK, "OK")
		})

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing UUID Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.Use(UserIDMiddleware())
		r.GET("/", func(c *gin.Context) {
			_, exists := c.Get(ctxconsts.UserIDKey)
			assert.False(t, exists)

			reqUid := c.Request.Context().Value(ctxconsts.UserIDKey)
			assert.Nil(t, reqUid)

			c.String(http.StatusOK, "OK")
		})

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
