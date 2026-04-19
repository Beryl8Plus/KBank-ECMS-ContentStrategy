-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schedules_active_window
ON schedules ("EFFECTIVE_FROM", "EFFECTIVE_UNTIL")
WHERE "IS_ACTIVE" = true AND "DELETED_AT" IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schedules_created_at_desc_active
ON schedules ("CREATED_AT" DESC)
WHERE "DELETED_AT" IS NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_schedules_created_at_desc_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_schedules_active_window;