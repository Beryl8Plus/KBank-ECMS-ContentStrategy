//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
)

// InitializeApp wires all dependencies to construct the Gin engine.
func InitializeApp(db *gorm.DB, rateLimit entity.RateLimit) (*gin.Engine, error) {
	wire.Build(ProviderSet)
	return nil, nil
}
