package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseHeaderMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		requestID     string
		handlerStatus int
	}{
		{name: "200 OK", requestID: "req-abc", handlerStatus: http.StatusOK},
		{name: "201 Created", requestID: "req-def", handlerStatus: http.StatusCreated},
		{name: "400 Bad Request", requestID: "", handlerStatus: http.StatusBadRequest},
		{name: "401 Unauthorized", requestID: "req-ghi", handlerStatus: http.StatusUnauthorized},
		{name: "403 Forbidden", requestID: "req-jkl", handlerStatus: http.StatusForbidden},
		{name: "404 Not Found", requestID: "req-mno", handlerStatus: http.StatusNotFound},
		{name: "422 Unprocessable Entity", requestID: "req-pqr", handlerStatus: http.StatusUnprocessableEntity},
		{name: "500 Internal Server Error", requestID: "req-stu", handlerStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := gin.New()
			r.Use(ResponseHeaderMiddleware())
			r.GET("/test", func(c *gin.Context) {
				c.Status(tt.handlerStatus)
			})

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			if tt.requestID != "" {
				req.Header.Set("requestID", tt.requestID)
			}

			r.ServeHTTP(w, req)

			resp := w.Result()
			assert.Equal(t, "application/json; charset=UTF-8", resp.Header.Get("Content-Type"))
			assert.Equal(t, tt.requestID, resp.Header.Get("Request-ID"))
			assert.Equal(t, strconv.Itoa(tt.handlerStatus), resp.Header.Get("Status-Code"))
			assert.Equal(t, http.StatusText(tt.handlerStatus), resp.Header.Get("Status-Msg"))
			assert.Equal(t, "Request-ID, Request-Time, Status-Code, Status-Msg", resp.Header.Get("Access-Control-Expose-Headers"))

			reqTime := resp.Header.Get("Request-Time")
			assert.NotEmpty(t, reqTime)
			_, parseErr := time.Parse("2006-01-02T15:04:05.000", reqTime)
			assert.NoError(t, parseErr, "Request-Time must follow format 2006-01-02T15:04:05.000")
		})
	}
}

func TestResponseHeaderMiddleware_ContentTypeNotOverriddenByHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	r := gin.New()
	r.Use(ResponseHeaderMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "application/json; charset=UTF-8", w.Result().Header.Get("Content-Type"))
	assert.Equal(t, "200", w.Result().Header.Get("Status-Code"))
	assert.Equal(t, "OK", w.Result().Header.Get("Status-Msg"))
}

func TestResponseHeaderMiddleware_AbortPreservesStatusHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	r := gin.New()
	r.Use(ResponseHeaderMiddleware())
	r.Use(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "401", w.Result().Header.Get("Status-Code"))
	assert.Equal(t, "Unauthorized", w.Result().Header.Get("Status-Msg"))
}
