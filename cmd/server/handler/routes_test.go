package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"kbank-ecms/cmd/server/handler"
	"kbank-ecms/internal/delivery/http/dto"
)

// stubService implements service.DeliveryService with no-op methods —
// enough to register routes against a real gin engine.
type stubService struct{}

func (stubService) GetPersonalizedContent(_ context.Context, _ *dto.CustomerRequest, _ string, _ []string) ([]dto.ContentResult, error) {
	return nil, nil
}
func (stubService) GetCacheKeys(_ context.Context) ([]string, error) { return nil, nil }
func (stubService) GetCacheValue(_ context.Context, _ string) (json.RawMessage, error) {
	return nil, nil
}
func (stubService) GetCacheStatus(_ context.Context) (bool, float64, error) { return false, 0, nil }
func (stubService) FlushCache(_ context.Context, _ []string, _ bool) error  { return nil }

func TestRegisterRoutes_AttachesExpectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.RegisterRoutes(r, stubService{})

	// Each entry hits an endpoint; we only assert non-404 (route registered).
	for _, ep := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/content-strategy/v1/purge_requests", ""},
		{http.MethodGet, "/api/content-strategy/v1/purge_requests/value?key=x", ""},
		{http.MethodPost, "/api/content-strategy/v1/purge_requests", `{"placements":["x"]}`},
	} {
		var req *http.Request
		if ep.body != "" {
			req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(ep.method, ep.path, nil)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s returned 404 — route missing", ep.method, ep.path)
		}
	}
}
