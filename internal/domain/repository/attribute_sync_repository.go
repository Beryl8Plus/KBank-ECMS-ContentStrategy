package repository

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// AttributeSyncRepository handles bulk attribute operations used by the sync job.
type AttributeSyncRepository interface {
	// BulkUpsertAttributes inserts or updates attributes matched on
	// (CLEN_SCHEMA_REGISTRY_ID, FIELD_NAME). On conflict it updates VALUE,
	// DISPLAY_NAME, SOURCE_SYSTEM, and resets IS_ACTIVE to true.
	// The ID field of each element is populated with the upserted row's ID.
	BulkUpsertAttributes(ctx context.Context, attrs []*entity.Attribute) error

	// DeactivateMissingAttributes sets IS_ACTIVE = false for every non-deleted
	// attribute whose ID is not in keepIDs. Used after a full sync to retire
	// attributes that no longer exist in the external schema.
	DeactivateMissingAttributes(ctx context.Context, keepIDs []uuid.UUID) error

	// FindAttributesByIDs returns attributes for the given IDs (non-deleted only).
	FindAttributesByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Attribute, error)
}
