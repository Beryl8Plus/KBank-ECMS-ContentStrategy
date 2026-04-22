package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"

	"github.com/google/uuid"
)

// AttributeRepository defines the contract for attribute-related database operations.
type AttributeRepository interface {
	// CreateAttribute persists a new attribute record.
	CreateAttribute(ctx context.Context, attribute *entity.Attribute) error

	// GetAttributeByID retrieves a single non-deleted attribute by its primary key.
	// Returns (nil, nil) when no record is found.
	GetAttributeByID(ctx context.Context, id uuid.UUID) (*entity.Attribute, error)

	// ListAttributesPaginated returns a page of non-deleted attributes ordered by
	// created_at descending together with the total count of matching records.
	ListAttributesPaginated(ctx context.Context, page, limit int) ([]*entity.Attribute, int64, error)

	// UpdateAttribute saves all fields of an existing attribute record.
	UpdateAttribute(ctx context.Context, attribute *entity.Attribute) error

	// DeleteAttribute soft-deletes the attribute with the given ID.
	DeleteAttribute(ctx context.Context, id uuid.UUID) error
}
