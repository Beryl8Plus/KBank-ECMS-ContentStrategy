package service

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// AttributeService encapsulates business logic for attribute management.
type AttributeService struct {
	repo domainrepo.AttributeRepository
}

// NewAttributeService creates a new AttributeService.
func NewAttributeService(repo domainrepo.AttributeRepository) *AttributeService {
	return &AttributeService{repo: repo}
}

// CreateAttribute persists a new attribute.
func (s *AttributeService) CreateAttribute(ctx context.Context, attribute *entity.Attribute) error {
	return s.repo.CreateAttribute(ctx, attribute)
}

// GetAttributeByID retrieves a single attribute by its ID.
// Returns (nil, nil) when not found.
func (s *AttributeService) GetAttributeByID(ctx context.Context, id uuid.UUID) (*entity.Attribute, error) {
	return s.repo.GetAttributeByID(ctx, id)
}

// ListAttributesPaginated returns a page of non-deleted attributes and the total record count.
// page and limit must be >= 1.
func (s *AttributeService) ListAttributesPaginated(ctx context.Context, page, limit int) ([]*entity.Attribute, int64, error) {
	return s.repo.ListAttributesPaginated(ctx, page, limit)
}

// UpdateAttribute saves the updated attribute.
func (s *AttributeService) UpdateAttribute(ctx context.Context, attribute *entity.Attribute) error {
	return s.repo.UpdateAttribute(ctx, attribute)
}

// DeleteAttribute soft-deletes an attribute by ID.
func (s *AttributeService) DeleteAttribute(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteAttribute(ctx, id)
}
