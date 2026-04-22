//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	domainservice "kbank-ecms/internal/domain/service"
)

// InitializeApp wires all dependencies to construct the delivery service and Gin engine.
func InitializeApp(
	db *gorm.DB,
	rateLimit entity.RateLimit,
	redisRepo domainrepo.RedisCacheRepository,
	evaluator domainservice.RuntimeEvaluator,
) (*App, func()) {
	wire.Build(ProviderSet)
	return nil, nil
}
