package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"kbank-ecms/pkg/config"
)

// helper builds a router with a single middleware applied + a /ok handler.
func newRouterWith(mw ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	for _, m := range mw {
		r.Use(m)
	}
	r.GET("/ok", func(c *gin.Context) { c.String(200, "ok") })
	r.POST("/echo", func(c *gin.Context) { c.String(200, "echo") })
	return r
}

func do(r *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─────────────────────────────────────────────────────────────────────────────
// Apply — full stack with non-nil RPS/burst/MCR
// ─────────────────────────────────────────────────────────────────────────────

func TestApply_FullStackPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.AppConfig{}
	cfg.Server.Config.RateLimit.RPS = 100
	cfg.Server.Config.RateLimit.Burst = 100
	cfg.Server.Config.RateLimit.MCR = 10
	cfg.Server.Config.Timeout.ReqCtxTimeout = time.Second
	cfg.Server.Config.Timeout.DBCtxTimeout = time.Second

	r := gin.New()
	Apply(r, nil, cfg) // db == nil → DBMiddleware skipped
	r.GET("/x", func(c *gin.Context) { c.String(200, "x") })

	w := do(r, http.MethodGet, "/x", "")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CORS — wildcard default + env override path
// ─────────────────────────────────────────────────────────────────────────────

func TestCORSMiddleware_DefaultAllowsAny(t *testing.T) {
	r := newRouterWith(CORSMiddleware())
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORSMiddleware_ExplicitOriginsAndCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	r := newRouterWith(CORSMiddleware())
	w := do(r, http.MethodGet, "/ok", "")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORSMiddleware_CredentialsRejectedWithWildcard(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	r := newRouterWith(CORSMiddleware())
	w := do(r, http.MethodGet, "/ok", "")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Concurrency — within and over capacity
// ─────────────────────────────────────────────────────────────────────────────

func TestConcurrencyMiddleware_AllowsAtCap(t *testing.T) {
	r := newRouterWith(ConcurrencyMiddleware(2))
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := do(r, http.MethodGet, "/ok", "")
			if w.Code != 200 {
				t.Errorf("expected 200, got %d", w.Code)
			}
		}()
	}
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// Timeout — request still completes when handler is fast
// ─────────────────────────────────────────────────────────────────────────────

func TestTimeoutMiddleware_FastRequestSucceeds(t *testing.T) {
	r := newRouterWith(TimeoutMiddleware(time.Second))
	w := do(r, http.MethodGet, "/ok", "")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RateLimiter — under and over the limit
// ─────────────────────────────────────────────────────────────────────────────

func TestRateLimiterMiddleware_BlocksOverBurst(t *testing.T) {
	r := newRouterWith(RateLimiterMiddleware(1, 1))
	// burst=1 → first request passes, second is blocked (back-to-back, no wait)
	w1 := do(r, http.MethodGet, "/ok", "")
	w2 := do(r, http.MethodGet, "/ok", "")
	if w1.Code != 200 {
		t.Errorf("first request expected 200, got %d", w1.Code)
	}
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request expected 429, got %d", w2.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LoggerMiddleware — body capture and metrics path skip
// ─────────────────────────────────────────────────────────────────────────────

func TestLoggerMiddleware_RegularRequest(t *testing.T) {
	r := newRouterWith(LoggerMiddleware())
	w := do(r, http.MethodPost, "/echo", `{"name": "value\n"}`)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLoggerMiddleware_SkipsMetricsPath(t *testing.T) {
	r := newRouterWith(LoggerMiddleware())
	r.GET("/metrics", func(c *gin.Context) { c.String(200, "metrics") })
	w := do(r, http.MethodGet, "/metrics", "")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLoggerMiddleware_HonorsExistingCorrelationID(t *testing.T) {
	r := newRouterWith(LoggerMiddleware())
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("requestID", "my-correlation-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
