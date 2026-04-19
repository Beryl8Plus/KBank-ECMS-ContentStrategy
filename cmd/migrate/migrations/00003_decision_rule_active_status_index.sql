-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_decision_rules_active_status
ON decision_rules ("STATUS")
WHERE "STATUS" = 'ACTIVE' AND "DELETED_AT" IS NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_decision_rules_active_status;
