package middleware

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterMiddleware creates a rate limiter based on requests per second and burst.
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
