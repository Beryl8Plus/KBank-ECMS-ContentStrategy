package repository

import (
	domainrepo "kbank-ecms/internal/domain/repository"

	"gorm.io/gorm"
)

// PostgresRepository implements domain repository.DatabaseRepository using GORM.
type PostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.DatabaseRepository = (*PostgresRepository)(nil)

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// AutoMigrate runs GORM auto-migration for the given models.
func (r *PostgresRepository) AutoMigrate(models ...interface{}) error {
	return r.db.AutoMigrate(models...)
}

// GetDB returns the underlying *gorm.DB instance.
func (r *PostgresRepository) GetDB() *gorm.DB {
	return r.db
}
