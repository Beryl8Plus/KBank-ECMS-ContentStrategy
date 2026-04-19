package util

import (
	"kbank-ecms/internal/domain/entity"
	"os"

	"github.com/goccy/go-yaml"
)

func LoadNewServiceRateLimit(path string) (entity.RateLimit, error) {
	var cfg entity.InboundConfig
	body, err := os.ReadFile(path)
	if err != nil {
		return entity.RateLimit{}, err
	}

	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return entity.RateLimit{}, err
	}

	if len(cfg.Server) == 0 {
		return entity.RateLimit{}, os.ErrNotExist
	}

	return cfg.Server[0].RateLimit, nil
}
