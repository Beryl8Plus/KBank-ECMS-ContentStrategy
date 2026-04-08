package repository

import (
	"context"

	domainrepo "kbank-ecms/internal/domain/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PermissionPostgresRepository implements domainrepo.PermissionRepository using GORM.
type PermissionPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.PermissionRepository = (*PermissionPostgresRepository)(nil)

// NewPermissionPostgresRepository creates a new PermissionPostgresRepository.
func NewPermissionPostgresRepository(db *gorm.DB) *PermissionPostgresRepository {
	return &PermissionPostgresRepository{db: db}
}

// HasPermission reports whether the user's active profile grants the given source+action.
// Joins: users → profile_permissions → permissions, applying soft-delete filters on each table.
func (r *PermissionPostgresRepository) HasPermission(ctx context.Context, userID uuid.UUID, source, action string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("profile_permissions AS pp").
		Joins("INNER JOIN permissions AS p ON p.id = pp.permission_id AND p.deleted_at IS NULL").
		Joins("INNER JOIN users AS u ON u.profile_id = pp.profile_id AND u.deleted_at IS NULL").
		Where("u.id = ? AND p.source = ? AND p.action = ? AND pp.deleted_at IS NULL AND u.is_active = TRUE",
			userID, source, action).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
