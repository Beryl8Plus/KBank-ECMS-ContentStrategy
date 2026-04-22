package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"

	"github.com/google/uuid"
)

// PlacementRepository defines the contract for placement-related database operations.
// All read methods preload the associated Channel.
type PlacementRepository interface {
	// CreatePlacement persists a new placement record.
	CreatePlacement(ctx context.Context, placement *entity.Placement) error

	// GetPlacementByID retrieves a single non-deleted placement by its primary key,
	// with the associated Channel preloaded. Returns (nil, nil) when no record is found.
	GetPlacementByID(ctx context.Context, id uuid.UUID) (*entity.Placement, error)

	// ListPlacementsPaginated returns a page of non-deleted placements ordered by
	// created_at descending together with the total count, with Channel preloaded.
	ListPlacementsPaginated(ctx context.Context, page, limit int) ([]*entity.Placement, int64, error)

	// UpdatePlacement saves all fields of an existing placement record.
	UpdatePlacement(ctx context.Context, placement *entity.Placement) error

	// DeletePlacement soft-deletes the placement with the given ID.
	DeletePlacement(ctx context.Context, id uuid.UUID) error
}
