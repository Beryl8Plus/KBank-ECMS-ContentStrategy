package service

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// PlacementService encapsulates business logic for placement management.
type PlacementService struct {
	repo domainrepo.PlacementRepository
}

// NewPlacementService creates a new PlacementService.
func NewPlacementService(repo domainrepo.PlacementRepository) *PlacementService {
	return &PlacementService{repo: repo}
}

// CreatePlacement persists a new placement.
func (s *PlacementService) CreatePlacement(ctx context.Context, placement *entity.Placement) error {
	return s.repo.CreatePlacement(ctx, placement)
}

// GetPlacementByID retrieves a single placement by its ID (with Channel preloaded).
// Returns (nil, nil) when not found.
func (s *PlacementService) GetPlacementByID(ctx context.Context, id uuid.UUID) (*entity.Placement, error) {
	return s.repo.GetPlacementByID(ctx, id)
}

// ListPlacementsPaginated returns a page of non-deleted placements and the total record count.
func (s *PlacementService) ListPlacementsPaginated(ctx context.Context, page, limit int) ([]*entity.Placement, int64, error) {
	return s.repo.ListPlacementsPaginated(ctx, page, limit)
}

// UpdatePlacement saves the updated placement.
func (s *PlacementService) UpdatePlacement(ctx context.Context, placement *entity.Placement) error {
	return s.repo.UpdatePlacement(ctx, placement)
}

// DeletePlacement soft-deletes a placement by ID.
func (s *PlacementService) DeletePlacement(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePlacement(ctx, id)
}
