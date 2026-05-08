package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/pkg/config"
)

// stubDeliveryService is a tiny no-op DeliveryService for ProvideRouter wiring tests.
type stubDeliveryService struct{}

func (stubDeliveryService) GetPersonalizedContent(_ context.Context, _ *dto.CustomerRequest, _ string, _ []string) ([]dto.ContentResult, error) {
	return nil, nil
}
func (stubDeliveryService) GetCacheKeys(_ context.Context) ([]string, error) { return nil, nil }
func (stubDeliveryService) GetCacheValue(_ context.Context, _ string) (json.RawMessage, error) {
	return nil, nil
}
func (stubDeliveryService) GetCacheStatus(_ context.Context) (bool, float64, error) {
	return false, 0, nil
}
func (stubDeliveryService) FlushCache(_ context.Context, _ []string, _ bool) error { return nil }

func TestProvideRouter_NilDepsBuildsEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.AppConfig{}
	cfg.Server.Config.RateLimit.RPS = 100
	cfg.Server.Config.RateLimit.Burst = 100
	cfg.Server.Config.RateLimit.MCR = 10

	got := ProvideRouter(cfg, nil, nil, stubDeliveryService{})
	if got == nil {
		t.Fatal("expected non-nil router")
	}
}
