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

// InitializeApp wires all dependencies to construct the delivery service and Gin engine.
func InitializeApp(
	db *gorm.DB,
	redisRepo *repository.RedisRepository,
	rateLimit entity.RateLimit,
) (*App, func()) {
	wire.Build(
		ProviderSet,
		wire.Bind(new(domainrepo.RedisCacheRepository), new(*repository.RedisRepository)),
	)
	return nil, nil
}
