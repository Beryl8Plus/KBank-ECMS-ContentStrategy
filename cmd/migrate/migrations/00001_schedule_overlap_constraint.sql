-- +goose NO TRANSACTION

-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'no_overlap_active_schedule_per_rule_placement'
  ) THEN
    ALTER TABLE schedules
    ADD CONSTRAINT no_overlap_active_schedule_per_rule_placement
    EXCLUDE USING gist (
      "DECISION_RULE_ID" WITH =,
      "PLACEMENT_ID" WITH =,
      tstzrange("EFFECTIVE_FROM", "EFFECTIVE_UNTIL") WITH &&
    ) WHERE ("IS_ACTIVE" = true AND "DELETED_AT" IS NULL);
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'no_overlap_active_schedule_per_rule_placement'
  ) THEN
    ALTER TABLE schedules DROP CONSTRAINT no_overlap_active_schedule_per_rule_placement;
  END IF;
END $$;
-- +goose StatementEnd
