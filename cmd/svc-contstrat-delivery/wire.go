//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/config"
)

// InitializeApp wires all dependencies to construct the delivery service and Gin engine.
func InitializeApp(
	cfg config.AppConfig,
	db *gorm.DB,
	redisRepo *repository.RedisRepository,
) (*App, func()) {
	wire.Build(
		ProviderSet,
		wire.Bind(new(domainrepo.RedisCacheRepository), new(*repository.RedisRepository)),
	)
	return nil, nil
}
