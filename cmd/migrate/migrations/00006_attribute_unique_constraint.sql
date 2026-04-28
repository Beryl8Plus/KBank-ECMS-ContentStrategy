-- +goose Up
-- Unique constraint for attribute sync upsert key: (schema_registry, field_name).
-- Partial index excludes soft-deleted rows so they don't block re-insertion.
CREATE UNIQUE INDEX IF NOT EXISTS uq_attributes_schema_field
    ON attributes (CLEN_SCHEMA_REGISTRY_ID, FIELD_NAME)
    WHERE DELETED_AT IS NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_attributes_schema_field;
