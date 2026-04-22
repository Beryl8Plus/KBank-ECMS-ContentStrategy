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

// PlacementPostgresRepository implements domainrepo.PlacementRepository using GORM.
type PlacementPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.PlacementRepository = (*PlacementPostgresRepository)(nil)

// NewPlacementPostgresRepository creates a new PlacementPostgresRepository.
func NewPlacementPostgresRepository(db *gorm.DB) *PlacementPostgresRepository {
	return &PlacementPostgresRepository{db: db}
}

// CreatePlacement persists a new placement record.
func (r *PlacementPostgresRepository) CreatePlacement(ctx context.Context, placement *entity.Placement) error {
	if err := r.db.WithContext(ctx).Create(placement).Error; err != nil {
		return fmt.Errorf("creating placement: %w", err)
	}
	return nil
}

// GetPlacementByID retrieves a single non-deleted placement by primary key with Channel preloaded.
// Returns (nil, nil) when no record is found.
func (r *PlacementPostgresRepository) GetPlacementByID(ctx context.Context, id uuid.UUID) (*entity.Placement, error) {
	var p entity.Placement
	err := r.db.WithContext(ctx).
		Preload("Channel").
		First(&p, `"ID" = ?`, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting placement by ID: %w", err)
	}
	return &p, nil
}

// ListPlacementsPaginated returns a page of non-deleted placements ordered by created_at
// descending with Channel preloaded, together with the total count.
func (r *PlacementPostgresRepository) ListPlacementsPaginated(ctx context.Context, page, limit int) ([]*entity.Placement, int64, error) {
	var placements []*entity.Placement
	var total int64

	base := r.db.WithContext(ctx).Model(&entity.Placement{})
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting placements: %w", err)
	}

	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).
		Preload("Channel").
		Order(`"CREATED_AT" DESC`).
		Limit(limit).
		Offset(offset).
		Find(&placements).Error; err != nil {
		return nil, 0, fmt.Errorf("listing placements paginated: %w", err)
	}

	return placements, total, nil
}

// UpdatePlacement saves all fields of an existing placement record.
func (r *PlacementPostgresRepository) UpdatePlacement(ctx context.Context, placement *entity.Placement) error {
	if err := r.db.WithContext(ctx).Save(placement).Error; err != nil {
		return fmt.Errorf("updating placement: %w", err)
	}
	return nil
}

// DeletePlacement soft-deletes the placement with the given ID.
func (r *PlacementPostgresRepository) DeletePlacement(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.Placement{}, `"ID" = ?`, id).Error; err != nil {
		return fmt.Errorf("deleting placement: %w", err)
	}
	return nil
}
