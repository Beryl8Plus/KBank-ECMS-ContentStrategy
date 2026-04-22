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

// ChannelPostgresRepository implements domainrepo.ChannelRepository using GORM.
type ChannelPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.ChannelRepository = (*ChannelPostgresRepository)(nil)

// NewChannelPostgresRepository creates a new ChannelPostgresRepository.
func NewChannelPostgresRepository(db *gorm.DB) *ChannelPostgresRepository {
	return &ChannelPostgresRepository{db: db}
}

// CreateChannel persists a new channel record.
func (r *ChannelPostgresRepository) CreateChannel(ctx context.Context, channel *entity.Channel) error {
	if err := r.db.WithContext(ctx).Create(channel).Error; err != nil {
		return fmt.Errorf("creating channel: %w", err)
	}
	return nil
}

// GetChannelByID retrieves a single non-deleted channel by primary key.
// Returns (nil, nil) when no record is found.
func (r *ChannelPostgresRepository) GetChannelByID(ctx context.Context, id uuid.UUID) (*entity.Channel, error) {
	var c entity.Channel
	err := r.db.WithContext(ctx).First(&c, `"ID" = ?`, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting channel by ID: %w", err)
	}
	return &c, nil
}

// ListChannelsPaginated returns a page of non-deleted channels ordered by created_at
// descending together with the total count. page and limit are 1-based and must be >= 1.
func (r *ChannelPostgresRepository) ListChannelsPaginated(ctx context.Context, page, limit int) ([]*entity.Channel, int64, error) {
	var channels []*entity.Channel
	var total int64

	base := r.db.WithContext(ctx).Model(&entity.Channel{})
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting channels: %w", err)
	}

	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).
		Order(`"CREATED_AT" DESC`).
		Limit(limit).
		Offset(offset).
		Find(&channels).Error; err != nil {
		return nil, 0, fmt.Errorf("listing channels paginated: %w", err)
	}

	return channels, total, nil
}

// UpdateChannel saves all fields of an existing channel record.
func (r *ChannelPostgresRepository) UpdateChannel(ctx context.Context, channel *entity.Channel) error {
	if err := r.db.WithContext(ctx).Save(channel).Error; err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}
	return nil
}

// DeleteChannel soft-deletes the channel with the given ID.
func (r *ChannelPostgresRepository) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.Channel{}, `"ID" = ?`, id).Error; err != nil {
		return fmt.Errorf("deleting channel: %w", err)
	}
	return nil
}
