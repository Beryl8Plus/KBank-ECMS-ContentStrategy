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

// AttributePostgresRepository implements domainrepo.AttributeRepository using GORM.
type AttributePostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.AttributeRepository = (*AttributePostgresRepository)(nil)

// NewAttributePostgresRepository creates a new AttributePostgresRepository.
func NewAttributePostgresRepository(db *gorm.DB) *AttributePostgresRepository {
	return &AttributePostgresRepository{db: db}
}

// CreateAttribute persists a new attribute record.
func (r *AttributePostgresRepository) CreateAttribute(ctx context.Context, attribute *entity.Attribute) error {
	if err := r.db.WithContext(ctx).Create(attribute).Error; err != nil {
		return fmt.Errorf("creating attribute: %w", err)
	}
	return nil
}

// GetAttributeByID retrieves a single non-deleted attribute by primary key.
// Returns (nil, nil) when no record is found.
func (r *AttributePostgresRepository) GetAttributeByID(ctx context.Context, id uuid.UUID) (*entity.Attribute, error) {
	var a entity.Attribute
	err := r.db.WithContext(ctx).First(&a, "\"ID\" = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting attribute by ID: %w", err)
	}
	return &a, nil
}

// ListAttributesPaginated returns a page of non-deleted attributes ordered by created_at
// descending together with the total count. page and limit are 1-based and must be >= 1.
func (r *AttributePostgresRepository) ListAttributesPaginated(ctx context.Context, page, limit int) ([]*entity.Attribute, int64, error) {
	var attributes []*entity.Attribute
	var total int64

	base := r.db.WithContext(ctx).Model(&entity.Attribute{})
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting attributes: %w", err)
	}

	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).
		Order("\"CREATED_AT\" DESC").
		Limit(limit).
		Offset(offset).
		Find(&attributes).Error; err != nil {
		return nil, 0, fmt.Errorf("listing attributes paginated: %w", err)
	}

	return attributes, total, nil
}

// UpdateAttribute saves all fields of an existing attribute record.
func (r *AttributePostgresRepository) UpdateAttribute(ctx context.Context, attribute *entity.Attribute) error {
	if err := r.db.WithContext(ctx).Save(attribute).Error; err != nil {
		return fmt.Errorf("updating attribute: %w", err)
	}
	return nil
}

// DeleteAttribute soft-deletes the attribute with the given ID.
func (r *AttributePostgresRepository) DeleteAttribute(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.Attribute{}, "\"ID\" = ?", id).Error; err != nil {
		return fmt.Errorf("deleting attribute: %w", err)
	}
	return nil
}
