package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"

	"github.com/google/uuid"
)

// CLENSchemaRegistryRepository abstracts read access to clen_schema_registry.
// Used by delivery to look up the field dictionary (SchemaDefinition) for a
// given schema, so per-request CLEN queries can include the full set of
// schema-declared fields for cache warming.
type CLENSchemaRegistryRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.CLENSchemaRegistry, error)
}
