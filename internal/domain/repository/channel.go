package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"

	"github.com/google/uuid"
)

// ChannelRepository defines the contract for channel-related database operations.
type ChannelRepository interface {
	// CreateChannel persists a new channel record.
	CreateChannel(ctx context.Context, channel *entity.Channel) error

	// GetChannelByID retrieves a single non-deleted channel by its primary key.
	// Returns (nil, nil) when no record is found.
	GetChannelByID(ctx context.Context, id uuid.UUID) (*entity.Channel, error)

	// ListChannelsPaginated returns a page of non-deleted channels ordered by
	// created_at descending together with the total count of matching records.
	ListChannelsPaginated(ctx context.Context, page, limit int) ([]*entity.Channel, int64, error)

	// UpdateChannel saves all fields of an existing channel record.
	UpdateChannel(ctx context.Context, channel *entity.Channel) error

	// DeleteChannel soft-deletes the channel with the given ID.
	DeleteChannel(ctx context.Context, id uuid.UUID) error
}
