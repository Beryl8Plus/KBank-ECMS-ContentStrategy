package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kbank-ecms/pkg/config"
)

func TestProvideCLENLeadConfig_FieldsPassedThrough(t *testing.T) {
	cfg := config.AppConfig{
		Server: config.ServerConfig{
			Config: config.ServiceConfig{
				CLENLead: config.CLENLeadAPIConfig{
					BaseURL:       "https://api.example.com",
					Path:          "/v1/leads/customer-level",
					APIKey:        "secret-key",
					AppIdentifier: "APP-001",
					Timeout:       3 * time.Second,
					RetryCount:    4,
					ExpireFilter:  "true",
				},
			},
		},
	}

	result := ProvideCLENLeadConfig(cfg)

	assert.Equal(t, "https://api.example.com", result.BaseURL)
	assert.Equal(t, "/v1/leads/customer-level", result.Path)
	assert.Equal(t, "secret-key", result.APIKey)
	assert.Equal(t, "APP-001", result.AppIdentifier)
	assert.Equal(t, 3*time.Second, result.Timeout)
	assert.Equal(t, 4, result.RetryCount)
	assert.Equal(t, "true", result.ExpireFilter)
}

func TestProvideCLENLeadConfig_NoneExpireFilterBecomesEmpty(t *testing.T) {
	cfg := config.AppConfig{
		Server: config.ServerConfig{
			Config: config.ServiceConfig{
				CLENLead: config.CLENLeadAPIConfig{
					BaseURL:      "https://api.example.com",
					APIKey:       "key",
					ExpireFilter: "none",
				},
			},
		},
	}

	result := ProvideCLENLeadConfig(cfg)

	assert.Equal(t, "https://api.example.com", result.BaseURL)
	assert.Equal(t, "key", result.APIKey)
	assert.Equal(t, "", result.ExpireFilter, `"none" must become empty string so CLEN omits the exp_f param`)
}

func TestProvideCLENLeadConfig_NoneExpireFilterCaseInsensitive(t *testing.T) {
	cfg := config.AppConfig{
		Server: config.ServerConfig{
			Config: config.ServiceConfig{
				CLENLead: config.CLENLeadAPIConfig{
					BaseURL:      "https://api.example.com",
					APIKey:       "key",
					ExpireFilter: "NONE",
				},
			},
		},
	}

	result := ProvideCLENLeadConfig(cfg)

	assert.Equal(t, "https://api.example.com", result.BaseURL)
	assert.Equal(t, "key", result.APIKey)
	assert.Equal(t, "", result.ExpireFilter)
}

func TestProvideCLENLeadConfig_NoneExpireFilterMixedCase(t *testing.T) {
	cfg := config.AppConfig{
		Server: config.ServerConfig{
			Config: config.ServiceConfig{
				CLENLead: config.CLENLeadAPIConfig{
					BaseURL:      "https://api.example.com",
					APIKey:       "key",
					ExpireFilter: "None",
				},
			},
		},
	}

	result := ProvideCLENLeadConfig(cfg)

	assert.Equal(t, "https://api.example.com", result.BaseURL)
	assert.Equal(t, "key", result.APIKey)
	assert.Equal(t, "", result.ExpireFilter)
}
