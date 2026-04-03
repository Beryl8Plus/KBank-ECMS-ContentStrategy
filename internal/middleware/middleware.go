package middleware

import (
	"bytes"
	"fmt"
	"io"
	"kbank-ecms/internal/model"
	"kbank-ecms/internal/util"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterMiddleware creates a rate limiter based on requests per second and burst
func RateLimiterMiddleware(rps int, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(429, gin.H{"error": "Too Many Requests"})
			return
		}
		c.Next()
	}
}

// ConcurrencyMiddleware limits the number of concurrent requests
func ConcurrencyMiddleware(maxConcurrent int) gin.HandlerFunc {
	semaphore := make(chan struct{}, maxConcurrent)
	return func(c *gin.Context) {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			c.Next()
		default:
			// Option 1: Reject if full
			// c.AbortWithStatusJSON(503, gin.H{"error": "Service Unavailable - Too many concurrent requests"})

			// Option 2: Wait (Blocking) - As per typical requirement, we might want to wait or reject.
			// Given "max_concurrent_requests" usually implies a hard limit on active processing.
			// Let's stick to blocking with a timeout or just blocking.
			// For simplicity and safety, let's block but with a context timeout check if needed.
			// However, standard "max concurrent" often just queues.
			// But to be strict:
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			c.Next()
		}
	}
}

// LoggerMiddleware logs request and response details
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

		util.LRequest(model.RequestLog{
			Service:        "Content-gateway",
			Level:          "REQUEST",
			Method:         c.Request.Method,
			URL:            c.Request.RequestURI,
			RequestPayload: reqPayload,
			ClientIP:       c.ClientIP(),
			CorrelationID:  c.GetHeader("requestID"),
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
