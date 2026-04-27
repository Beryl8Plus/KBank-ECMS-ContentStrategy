package middleware

import (
	"os"
	"slices"
	"strings"
	"time"

	gincontribcors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a CORS handler using gin-contrib/cors with safe,
// environment-configurable defaults. Set `CORS_ALLOWED_ORIGINS` to a
// comma-separated list of allowed origins. If unset, fallback is "*" but
// credentials are disabled by default.
//
// To allow credentials (cookies, Authorization headers automatically sent by
// browsers), set `CORS_ALLOW_CREDENTIALS=true` and ensure `CORS_ALLOWED_ORIGINS`
// contains explicit origins (not "*").
func CORSMiddleware() gin.HandlerFunc {
	allowed := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	var origins []string
	if allowed != "" {
		for _, o := range strings.Split(allowed, ",") {
			if s := strings.TrimSpace(o); s != "" {
				origins = append(origins, s)
			}
		}
	} else {
		origins = []string{"*"}
	}

	cfg := gincontribcors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id", "X-Request-Id"},
		ExposeHeaders:    []string{"Request-ID", "Request-Time", "Status-Code", "Status-Msg"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: false,
	}

	// Enable credentials only when explicitly requested and a non-wildcard
	// origin list is provided. Never enable credentials when AllowOrigins
	// contains "*".
	if strings.ToLower(os.Getenv("CORS_ALLOW_CREDENTIALS")) == "true" {
		if slices.Contains(cfg.AllowOrigins, "*") {
			// Unsafe to allow credentials with wildcard origin. Keep
			// credentials disabled and rely on explicit origin config.
			return gincontribcors.New(cfg)
		}
		cfg.AllowCredentials = true
	}

	return gincontribcors.New(cfg)
}
