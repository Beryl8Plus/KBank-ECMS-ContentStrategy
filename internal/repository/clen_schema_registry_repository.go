package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// CLENSchemaRegistryPostgresRepository implements
// domainrepo.CLENSchemaRegistryRepository using GORM.
type CLENSchemaRegistryPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.CLENSchemaRegistryRepository = (*CLENSchemaRegistryPostgresRepository)(nil)

// NewCLENSchemaRegistryPostgresRepository creates a new repository.
func NewCLENSchemaRegistryPostgresRepository(db *gorm.DB) *CLENSchemaRegistryPostgresRepository {
	return &CLENSchemaRegistryPostgresRepository{db: db}
}

// GetByID returns the schema registry row for the given ID, or (nil, nil)
// when not found. Errors only on transport/SQL failure.
func (r *CLENSchemaRegistryPostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.CLENSchemaRegistry, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	var schema entity.CLENSchemaRegistry
	err := r.db.WithContext(ctx).Where(`"ID" = ?`, id).First(&schema).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting clen_schema_registry by id: %w", err)
	}
	return &schema, nil
}
