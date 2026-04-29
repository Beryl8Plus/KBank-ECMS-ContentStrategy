package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// UserPostgresRepository implements domainrepo.UserRepository.
type UserPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.UserRepository = (*UserPostgresRepository)(nil)

// NewUserPostgresRepository creates a new UserPostgresRepository.
func NewUserPostgresRepository(db *gorm.DB) *UserPostgresRepository {
	return &UserPostgresRepository{db: db}
}

// FindByIDs returns Users matching the given IDs.
func (r *UserPostgresRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []*entity.User
	err := r.db.WithContext(ctx).
		Select(`"ID"`, `"NAME_TH"`, `"NAME_EN"`).
		Where(`"ID" IN ?`, ids).
		Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("finding users by ids: %w", err)
	}
	return users, nil
}
