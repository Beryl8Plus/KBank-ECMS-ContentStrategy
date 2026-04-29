package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gorm.io/gorm"
)

func TestDBMiddleware(t *testing.T) {
	// Create a dummy GORM DB instance.
	// We don't need a real connection because we're just testing if it's placed into the context correctly.
	dummyDB := &gorm.DB{
		Config:    &gorm.Config{},
		Statement: &gorm.Statement{},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a dummy request to avoid panics on c.Request.WithContext
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)

	handler := DBMiddleware(dummyDB, 10*time.Second)
	handler(c)

	// 1. Verify the DB was set in the Gin context
	ginCtxDB, exists := c.Get(string(ctxconsts.DBKey))
	assert.True(t, exists)
	assert.NotNil(t, ginCtxDB)

	// 2. Verify the DB was set in the standard request context
	reqCtxDB := c.Request.Context().Value(ctxconsts.DBKey)
	assert.NotNil(t, reqCtxDB)

	// The stored DB instance should have a context with a deadline set
	storedDB, ok := reqCtxDB.(*gorm.DB)
	assert.True(t, ok)

	deadline, hasDeadline := storedDB.Statement.Context.Deadline()
	assert.True(t, hasDeadline)
	// We set a 10s timeout, so the deadline should be roughly 10s from now
	assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, 100*time.Millisecond)
}
