package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	"kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"
)

// ExternalAttributeSchema is one field entry fetched from the CLEN external API.
type ExternalAttributeSchema struct {
	ClenSchemaRegistryID uuid.UUID
	FieldName            string
	DisplayName          string
	DataType             enums.AttributeDataType
	SourceSystem         string
	// Values holds the allowed options as jsonb (e.g. ["GOLD","SILVER"] or
	// [{"value":"GOLD","label":"ทอง"}]).
	Values datatypes.JSON
}

// ExternalAttributeAPIClient is the contract for fetching schemas from CLEN.
// Implement this interface with the actual HTTP client when the endpoint is available.
type ExternalAttributeAPIClient interface {
	FetchAllAttributes(ctx context.Context) ([]*ExternalAttributeSchema, error)
}

// AttributeSyncService pulls the latest attribute schema from CLEN and keeps
// the local attributes table in sync.
type AttributeSyncService struct {
	repo   repository.AttributeSyncRepository
	client ExternalAttributeAPIClient
}

// NewAttributeSyncService creates a new AttributeSyncService.
func NewAttributeSyncService(
	repo repository.AttributeSyncRepository,
	client ExternalAttributeAPIClient,
) *AttributeSyncService {
	return &AttributeSyncService{repo: repo, client: client}
}

// Sync fetches all attributes from the external API, upserts them locally,
// and deactivates any attribute that is no longer present in the schema.
func (s *AttributeSyncService) Sync(ctx context.Context) error {
	schemas, err := s.client.FetchAllAttributes(ctx)
	if err != nil {
		return fmt.Errorf("fetch external attribute schema: %w", err)
	}

	attrs := make([]*entity.Attribute, 0, len(schemas))
	for _, sc := range schemas {
		attrs = append(attrs, &entity.Attribute{
			ClenSchemaRegistryID: sc.ClenSchemaRegistryID,
			FieldName:            sc.FieldName,
			DisplayName:          sc.DisplayName,
			DataType:             sc.DataType,
			SourceSystem:         sc.SourceSystem,
			Value:                sc.Values,
			IsActive:             true,
		})
	}

	if err := s.repo.BulkUpsertAttributes(ctx, attrs); err != nil {
		return fmt.Errorf("bulk upsert: %w", err)
	}

	// Collect IDs populated by RETURNING after upsert.
	keepIDs := make([]uuid.UUID, 0, len(attrs))
	for _, a := range attrs {
		if a.ID != uuid.Nil {
			keepIDs = append(keepIDs, a.ID)
		}
	}

	if err := s.repo.DeactivateMissingAttributes(ctx, keepIDs); err != nil {
		return fmt.Errorf("deactivate missing: %w", err)
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "ATTRIBUTE-SYNC",
		Level:   "INFO",
		Message: fmt.Sprintf("sync complete: upserted=%d active_kept=%d", len(attrs), len(keepIDs)),
	})
	return nil
}
