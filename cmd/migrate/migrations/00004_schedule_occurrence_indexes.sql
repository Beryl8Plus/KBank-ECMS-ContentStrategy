-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
  -- Unique index for idempotent upsert via ON CONFLICT (SCHEDULE_ID, OCCURRENCE_START, OCCURRENCE_END)
  IF NOT EXISTS (
    SELECT 1
    FROM   pg_indexes
    WHERE  tablename  = 'schedule_occurrences'
    AND    indexname  = 'idx_occurrence_schedule_start_end'
  ) THEN
    CREATE UNIQUE INDEX idx_occurrence_schedule_start_end
      ON schedule_occurrences ("SCHEDULE_ID", "OCCURRENCE_START", "OCCURRENCE_END")
      WHERE "DELETED_AT" IS NULL;
  END IF;

  -- Performance index for the active-at query:
  --   WHERE STATUS = 'ACTIVE' AND OCCURRENCE_START <= :at AND OCCURRENCE_END > :at
  IF NOT EXISTS (
    SELECT 1
    FROM   pg_indexes
    WHERE  tablename  = 'schedule_occurrences'
    AND    indexname  = 'idx_occurrence_active_window'
  ) THEN
    CREATE INDEX idx_occurrence_active_window
      ON schedule_occurrences ("OCCURRENCE_START", "OCCURRENCE_END")
      WHERE "STATUS" = 'ACTIVE' AND "DELETED_AT" IS NULL;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_indexes WHERE tablename = 'schedule_occurrences' AND indexname = 'idx_occurrence_schedule_start_end'
  ) THEN
    DROP INDEX idx_occurrence_schedule_start_end;
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_indexes WHERE tablename = 'schedule_occurrences' AND indexname = 'idx_occurrence_active_window'
  ) THEN
    DROP INDEX idx_occurrence_active_window;
  END IF;
END $$;
-- +goose StatementEnd
