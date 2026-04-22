package service

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// ChannelService encapsulates business logic for channel management.
type ChannelService struct {
	repo domainrepo.ChannelRepository
}

// NewChannelService creates a new ChannelService.
func NewChannelService(repo domainrepo.ChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

// CreateChannel persists a new channel.
func (s *ChannelService) CreateChannel(ctx context.Context, channel *entity.Channel) error {
	return s.repo.CreateChannel(ctx, channel)
}

// GetChannelByID retrieves a single channel by its ID.
// Returns (nil, nil) when not found.
func (s *ChannelService) GetChannelByID(ctx context.Context, id uuid.UUID) (*entity.Channel, error) {
	return s.repo.GetChannelByID(ctx, id)
}

// ListChannelsPaginated returns a page of non-deleted channels and the total record count.
func (s *ChannelService) ListChannelsPaginated(ctx context.Context, page, limit int) ([]*entity.Channel, int64, error) {
	return s.repo.ListChannelsPaginated(ctx, page, limit)
}

// UpdateChannel saves the updated channel.
func (s *ChannelService) UpdateChannel(ctx context.Context, channel *entity.Channel) error {
	return s.repo.UpdateChannel(ctx, channel)
}

// DeleteChannel soft-deletes a channel by ID.
func (s *ChannelService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteChannel(ctx, id)
}
