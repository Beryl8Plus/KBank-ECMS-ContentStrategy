-- +goose Up
CREATE TABLE IF NOT EXISTS decision_rule_id_sequences (
    YEAR_MONTH TEXT    NOT NULL,
    LAST_SEQ   INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT pk_decision_rule_id_seq PRIMARY KEY (YEAR_MONTH)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION next_decision_rule_id()
RETURNS TEXT
LANGUAGE plpgsql
AS $$
DECLARE
    v_year_month TEXT;
    v_seq        INTEGER;
BEGIN
    v_year_month := TO_CHAR(NOW() AT TIME ZONE 'UTC', 'YYYYMM');
    INSERT INTO decision_rule_id_sequences (YEAR_MONTH, LAST_SEQ)
    VALUES (v_year_month, 1)
    ON CONFLICT (YEAR_MONTH) DO UPDATE
        SET LAST_SEQ = decision_rule_id_sequences.LAST_SEQ + 1
    RETURNING LAST_SEQ INTO v_seq;
    RETURN 'RS-' || v_year_month || '-' || LPAD(v_seq::TEXT, 4, '0');
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS next_decision_rule_id();
DROP TABLE IF EXISTS decision_rule_id_sequences;
