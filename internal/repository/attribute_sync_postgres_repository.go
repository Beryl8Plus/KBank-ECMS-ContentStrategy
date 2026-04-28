package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// AttributeSyncPostgresRepository implements domainrepo.AttributeSyncRepository.
type AttributeSyncPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.AttributeSyncRepository = (*AttributeSyncPostgresRepository)(nil)

// NewAttributeSyncPostgresRepository creates a new AttributeSyncPostgresRepository.
func NewAttributeSyncPostgresRepository(db *gorm.DB) *AttributeSyncPostgresRepository {
	return &AttributeSyncPostgresRepository{db: db}
}

// BulkUpsertAttributes inserts or updates attributes in batches of 500.
// On conflict on (CLEN_SCHEMA_REGISTRY_ID, FIELD_NAME) it updates VALUE,
// DISPLAY_NAME, SOURCE_SYSTEM, and resets IS_ACTIVE to true.
// Each element's ID is populated via the RETURNING clause.
func (r *AttributeSyncPostgresRepository) BulkUpsertAttributes(ctx context.Context, attrs []*entity.Attribute) error {
	if len(attrs) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{
					{Name: "CLEN_SCHEMA_REGISTRY_ID"},
					{Name: "FIELD_NAME"},
				},
				DoUpdates: clause.AssignmentColumns([]string{
					"VALUE", "DISPLAY_NAME", "SOURCE_SYSTEM", "IS_ACTIVE", "UPDATED_AT",
				}),
			},
			clause.Returning{Columns: []clause.Column{{Name: "ID"}}},
		).
		CreateInBatches(attrs, 500).Error
	if err != nil {
		return fmt.Errorf("bulk upsert attributes: %w", err)
	}
	return nil
}

// DeactivateMissingAttributes sets IS_ACTIVE = false for every active, non-deleted
// attribute whose ID is not in keepIDs.
func (r *AttributeSyncPostgresRepository) DeactivateMissingAttributes(ctx context.Context, keepIDs []uuid.UUID) error {
	if len(keepIDs) == 0 {
		// Safety guard: never deactivate everything when sync returned nothing.
		return nil
	}
	err := r.db.WithContext(ctx).
		Model(&entity.Attribute{}).
		Where(`"DELETED_AT" IS NULL AND "IS_ACTIVE" = true AND "ID" NOT IN ?`, keepIDs).
		Updates(map[string]any{"IS_ACTIVE": false}).Error
	if err != nil {
		return fmt.Errorf("deactivate missing attributes: %w", err)
	}
	return nil
}

// FindAttributesByIDs returns non-deleted attributes for the given IDs.
func (r *AttributeSyncPostgresRepository) FindAttributesByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Attribute, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var attrs []*entity.Attribute
	if err := r.db.WithContext(ctx).
		Where(`"ID" IN ?`, ids).
		Find(&attrs).Error; err != nil {
		return nil, fmt.Errorf("find attributes by ids: %w", err)
	}
	return attrs, nil
}
