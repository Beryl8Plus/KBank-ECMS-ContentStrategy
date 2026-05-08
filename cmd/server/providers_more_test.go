package main

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	httpclient "kbank-ecms/internal/http/client"
)

// helper that builds a noop fx.Lifecycle for direct provider invocation.
func newLC(t *testing.T) fx.Lifecycle {
	t.Helper()
	return fxtest.NewLifecycle(t)
}

func TestProvideCLENLeadClient_NilWhenConfigEmpty(t *testing.T) {
	got := ProvideCLENLeadClient(httpclient.CLENLeadConfig{})
	if got != nil {
		t.Errorf("expected nil client for empty config, got %v", got)
	}
}

func TestProvideCLENCustomerProfileConfig_DefaultRetry(t *testing.T) {
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_RETRIES", "")
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_TIMEOUT", "")
	cfg := ProvideCLENCustomerProfileConfig()
	if cfg.RetryCount != 2 {
		t.Errorf("expected RetryCount=2 when env unset, got %d", cfg.RetryCount)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", cfg.Timeout)
	}
}

func TestProvideCLENCustomerProfileConfig_HonorsEnv(t *testing.T) {
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_RETRIES", "7")
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_TIMEOUT", "12s")
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_BASE_URL", "https://x")
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_KEY", "k")
	cfg := ProvideCLENCustomerProfileConfig()
	if cfg.RetryCount != 7 {
		t.Errorf("RetryCount = %d", cfg.RetryCount)
	}
	if cfg.Timeout != 12*time.Second {
		t.Errorf("Timeout = %v", cfg.Timeout)
	}
	if cfg.BaseURL != "https://x" || cfg.APIKey != "k" {
		t.Errorf("env not propagated: %+v", cfg)
	}
}

func TestProvideCLENCustomerProfileConfig_BadRetryFallsBack(t *testing.T) {
	t.Setenv("CLEN_CUSTOMER_PROFILE_API_RETRIES", "not-a-number")
	cfg := ProvideCLENCustomerProfileConfig()
	if cfg.RetryCount != 2 {
		t.Errorf("expected fallback retry=2, got %d", cfg.RetryCount)
	}
}

func TestProvideCLENCustomerProfileClient_NilWhenConfigEmpty(t *testing.T) {
	got := ProvideCLENCustomerProfileClient(httpclient.CLENCustomerProfileConfig{})
	if got != nil {
		t.Errorf("expected nil client for empty config, got %v", got)
	}
}

func TestProvideCustomerProfileEnrichConfig_DefaultTTL(t *testing.T) {
	t.Setenv("CLEN_CUSTOMER_PROFILE_CACHE_TTL", "")
	got := ProvideCustomerProfileEnrichConfig()
	if got.CacheTTL != 5*time.Minute {
		t.Errorf("expected default 5m, got %v", got.CacheTTL)
	}
}

func TestProvideCustomerProfileEnrichConfig_HonorsEnv(t *testing.T) {
	t.Setenv("CLEN_CUSTOMER_PROFILE_CACHE_TTL", "30s")
	got := ProvideCustomerProfileEnrichConfig()
	if got.CacheTTL != 30*time.Second {
		t.Errorf("got %v", got.CacheTTL)
	}
}

func TestParseDurationEnv(t *testing.T) {
	t.Setenv("MY_TEST_DURATION_KEY", "")
	if got := parseDurationEnv("MY_TEST_DURATION_KEY", 7*time.Second); got != 7*time.Second {
		t.Errorf("empty env: got %v", got)
	}
	t.Setenv("MY_TEST_DURATION_KEY", "1500ms")
	if got := parseDurationEnv("MY_TEST_DURATION_KEY", 7*time.Second); got != 1500*time.Millisecond {
		t.Errorf("set: got %v", got)
	}
	t.Setenv("MY_TEST_DURATION_KEY", "garbage")
	if got := parseDurationEnv("MY_TEST_DURATION_KEY", 7*time.Second); got != 7*time.Second {
		t.Errorf("garbage falls back: got %v", got)
	}
}

func TestProvideRuntimeEvaluator(t *testing.T) {
	if ProvideRuntimeEvaluator() == nil {
		t.Fatal("expected non-nil evaluator")
	}
}

func TestProvideCacheMemory_RegistersStopHooks(t *testing.T) {
	lc := newLC(t)
	mem := ProvideCacheMemory(lc)
	if mem == nil || mem.Schedules == nil || mem.DecisionRule == nil ||
		mem.VersionHashes == nil || mem.LastSync == nil {
		t.Fatal("expected fully populated MemoryCache")
	}
	// drive lifecycle to exercise the OnStop hook
	if err := lc.(interface {
		Start(context.Context) error
	}).Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := lc.(interface {
		Stop(context.Context) error
	}).Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
