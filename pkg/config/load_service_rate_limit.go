package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

func LoadNewServiceRateLimit(path string) (RateLimitConfig, error) {
	var cfg InboundConfig
	body, err := os.ReadFile(path)
	if err != nil {
		return RateLimitConfig{}, err
	}

	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return RateLimitConfig{}, err
	}

	if len(cfg.Server) == 0 {
		return RateLimitConfig{}, os.ErrNotExist
	}

	return cfg.Server[0].RateLimit, nil
}
