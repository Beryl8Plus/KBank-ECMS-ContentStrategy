package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type responseHeaderWriter struct {
	gin.ResponseWriter
}

func (w *responseHeaderWriter) WriteHeaderNow() {
	if !w.Written() {
		w.setStatusHeaders()
	}
	w.ResponseWriter.WriteHeaderNow()
}

func (w *responseHeaderWriter) Write(b []byte) (int, error) {
	if !w.Written() {
		w.setStatusHeaders()
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseHeaderWriter) setStatusHeaders() {
	code := w.Status()
	w.Header().Set("Status-Code", strconv.Itoa(code))
	w.Header().Set("Status-Msg", http.StatusText(code))
}

// ResponseHeaderMiddleware sets standard response headers on every request.
func ResponseHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=UTF-8")
		c.Header("Request-ID", c.GetHeader("requestID"))
		c.Header("Request-Time", time.Now().Format("2006-01-02T15:04:05.000"))
		c.Header("Access-Control-Expose-Headers", "Request-ID, Request-Time, Status-Code, Status-Msg")

		c.Writer = &responseHeaderWriter{ResponseWriter: c.Writer}
		c.Next()

		// For responses with no body (e.g. c.Status only), Gin flushes headers
		// via c.writermem.WriteHeaderNow() after c.Next() returns, bypassing the
		// wrapped writer. Set status headers here so they are included in that flush.
		if !c.Writer.Written() {
			code := c.Writer.Status()
			c.Header("Status-Code", strconv.Itoa(code))
			c.Header("Status-Msg", http.StatusText(code))
		}
	}
}
