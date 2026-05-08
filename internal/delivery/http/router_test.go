package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"kbank-ecms/pkg/config"
)

func TestInitNewRouter_NilDepsAttachesObservability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.AppConfig{}
	cfg.Server.Config.RateLimit.RPS = 100
	cfg.Server.Config.RateLimit.Burst = 100
	cfg.Server.Config.RateLimit.MCR = 10
	r := InitNewRouter(cfg, nil, nil)
	if r == nil {
		t.Fatal("router must not be nil")
	}

	for _, tt := range []struct {
		path string
		want int
	}{
		{"/metrics", http.StatusOK},
		{"/healthz", http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.path, w.Code, tt.want)
		}
	}
}
