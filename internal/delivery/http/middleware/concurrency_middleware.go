package middleware

import "github.com/gin-gonic/gin"

// ConcurrencyMiddleware limits the number of concurrent requests.
func ConcurrencyMiddleware(maxConcurrent int) gin.HandlerFunc {
	semaphore := make(chan struct{}, maxConcurrent)
	return func(c *gin.Context) {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			c.Next()
		default:
			// Blocking wait when all slots are occupied.
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			c.Next()
		}
	}
}
