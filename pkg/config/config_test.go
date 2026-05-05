package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_CLENLeadDefaults(t *testing.T) {
	// Unset these so cleanenv falls back to YAML / env-default values.
	// t.Setenv(k, "") would set them to empty string, which cleanenv tries
	// to parse and fails for non-string types (time.Duration, int).
	for _, k := range []string{
		"CLEN_LEAD_API_BASE_URL", "CLEN_LEAD_API_PATH",
		"CLEN_LEAD_API_KEY", "CLEN_LEAD_APP_ID",
		"CLEN_LEAD_API_TIMEOUT", "CLEN_LEAD_API_RETRIES", "CLEN_LEAD_EXP_F",
	} {
		if prev, ok := os.LookupEnv(k); ok {
			t.Cleanup(func() { os.Setenv(k, prev) })
			os.Unsetenv(k)
		}
	}

	cfg, err := LoadConfig("../../configs/delivery.yaml")
	require.NoError(t, err)

	assert.Equal(t, 5*time.Second, cfg.Server.Config.CLENLead.Timeout)
	assert.Equal(t, 2, cfg.Server.Config.CLENLead.RetryCount)
	assert.Equal(t, "true", cfg.Server.Config.CLENLead.ExpireFilter)
}

func TestLoadConfig_CLENLeadEnvOverride(t *testing.T) {
	t.Setenv("CLEN_LEAD_API_BASE_URL", "https://api.example.com")
	t.Setenv("CLEN_LEAD_API_KEY", "secret-key")
	t.Setenv("CLEN_LEAD_API_RETRIES", "5")
	t.Setenv("CLEN_LEAD_EXP_F", "false")

	cfg, err := LoadConfig("../../configs/delivery.yaml")
	require.NoError(t, err)

	assert.Equal(t, "https://api.example.com", cfg.Server.Config.CLENLead.BaseURL)
	assert.Equal(t, "secret-key", cfg.Server.Config.CLENLead.APIKey)
	assert.Equal(t, 5, cfg.Server.Config.CLENLead.RetryCount)
	assert.Equal(t, "false", cfg.Server.Config.CLENLead.ExpireFilter)
}
