-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
  -- Drop the old partial unique index (WHERE DELETED_AT IS NULL) which prevented GORM's
  -- ON CONFLICT clause from matching it (PostgreSQL requires the exact WHERE predicate
  -- in ON CONFLICT to resolve a partial index, which GORM does not support).
  -- All occurrence deletes use hard delete (Unscoped), so the partial predicate is unnecessary.
  IF EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE tablename = 'schedule_occurrences'
    AND   indexname = 'idx_occurrence_schedule_start_end'
  ) THEN
    DROP INDEX idx_occurrence_schedule_start_end;
  END IF;

  -- Re-create as a non-partial unique index so ON CONFLICT works correctly.
  CREATE UNIQUE INDEX idx_occurrence_schedule_start_end
    ON schedule_occurrences ("SCHEDULE_ID", "OCCURRENCE_START", "OCCURRENCE_END");
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE tablename = 'schedule_occurrences'
    AND   indexname = 'idx_occurrence_schedule_start_end'
  ) THEN
    DROP INDEX idx_occurrence_schedule_start_end;
  END IF;

  CREATE UNIQUE INDEX idx_occurrence_schedule_start_end
    ON schedule_occurrences ("SCHEDULE_ID", "OCCURRENCE_START", "OCCURRENCE_END")
    WHERE "DELETED_AT" IS NULL;
END $$;
-- +goose StatementEnd
