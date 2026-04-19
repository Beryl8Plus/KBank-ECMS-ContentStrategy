package middleware

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/ctxconsts"
)

// LoggerMiddleware logs request and response details.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for metrics
		if c.Request.URL.Path == "/metrics" && c.Request.Method == "GET" {
			c.Next()
			return
		}

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Log Request
		reqPayload := string(requestBody)
		if reqPayload != "" {
			reqPayload = strings.ReplaceAll(reqPayload, "\n", "")
			reqPayload = strings.ReplaceAll(reqPayload, "\r", "")
			reqPayload = strings.ReplaceAll(reqPayload, " ", "")
		}

		correlationID := c.GetHeader("requestID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Propagate correlation ID through the request context so downstream
		// callers (e.g. the gRPC client) can forward it without re-reading headers.
		c.Request = c.Request.WithContext(ctxconsts.SetCorrelationID(c.Request.Context(), correlationID))

		logger.LRequest(c.Request.Context(), entity.RequestLog{
			Service:        "Content-gateway",
			Level:          "REQUEST",
			Method:         c.Request.Method,
			URL:            c.Request.RequestURI,
			RequestPayload: reqPayload,
			ClientIP:       c.ClientIP(),
			CorrelationID:  correlationID,
			Headers:        fmt.Sprintf("%v", c.Request.Header),
		})

		// Custom response writer to capture response body
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
