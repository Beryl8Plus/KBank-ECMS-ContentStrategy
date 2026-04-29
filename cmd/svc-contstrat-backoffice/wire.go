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

// InitializeApp wires all dependencies and returns the Application bundle
// (Gin engine + background occurrence worker).
func InitializeApp(cfg config.AppConfig, db *gorm.DB, redisCache *repository.RedisRepository) (*Application, error) {
	wire.Build(
		ProviderSet,
		wire.Bind(new(domainrepo.RedisCacheRepository), new(*repository.RedisRepository)),
	)
	return nil, nil
}
