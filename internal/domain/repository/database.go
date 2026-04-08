package repository

import "gorm.io/gorm"

// DatabaseRepository defines the contract for database operations.
type DatabaseRepository interface {
	// AutoMigrate runs GORM auto-migration for the given models.
	AutoMigrate(models ...interface{}) error

	// GetDB returns the underlying *gorm.DB instance for query building.
	GetDB() *gorm.DB
}
