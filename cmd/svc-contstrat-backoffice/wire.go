//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/repository"
)

// InitializeApp wires all dependencies and returns the Application bundle
// (Gin engine + background occurrence worker).
func InitializeApp(db *gorm.DB, redisCache *repository.RedisRepository, rateLimit entity.RateLimit) (*Application, error) {
	wire.Build(
		ProviderSet,
		wire.Bind(new(domainrepo.RedisCacheRepository), new(*repository.RedisRepository)),
	)
	return nil, nil
}
