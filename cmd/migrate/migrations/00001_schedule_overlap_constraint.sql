-- +goose NO TRANSACTION

-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'no_overlap_active_schedule_per_placement'
  ) THEN
    ALTER TABLE schedules
    ADD CONSTRAINT no_overlap_active_schedule_per_placement
    EXCLUDE USING gist (
      placement_id WITH =,
      tstzrange(effective_from, effective_until) WITH &&
    ) WHERE (is_active = true AND deleted_at IS NULL);
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'no_overlap_active_schedule_per_placement'
  ) THEN
    ALTER TABLE schedules DROP CONSTRAINT no_overlap_active_schedule_per_placement;
  END IF;
END $$;
-- +goose StatementEnd
