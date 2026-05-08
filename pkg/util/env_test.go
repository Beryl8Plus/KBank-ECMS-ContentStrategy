package util_test

import (
	"testing"

	"kbank-ecms/pkg/util"
)

func TestGetEnvWithDefault(t *testing.T) {
	t.Setenv("TEST_ENV_DEFAULT_KEY", "from-env")
	if got := util.GetEnvWithDefault("TEST_ENV_DEFAULT_KEY", "fallback"); got != "from-env" {
		t.Errorf("expected env value, got %q", got)
	}
	if got := util.GetEnvWithDefault("TEST_ENV_DEFAULT_KEY_MISSING", "fallback"); got != "fallback" {
		t.Errorf("expected fallback, got %q", got)
	}
}
