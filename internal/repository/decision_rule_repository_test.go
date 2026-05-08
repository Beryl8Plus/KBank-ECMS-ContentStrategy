package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// GetDecisionRuleByScheduleID
// ─────────────────────────────────────────────────────────────────────────────

func TestDecisionRule_GetByScheduleID_ScheduleNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT .* FROM schedules`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	r := NewDecisionRulePostgresRepository(db)
	got, err := r.GetDecisionRuleByScheduleID(context.Background(), uuid.New())
	if err != nil || got != nil {
		t.Errorf("not-found schedule: got=%v err=%v", got, err)
	}
}

func TestDecisionRule_GetByScheduleID_ScheduleQueryError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT .* FROM schedules`).
		WillReturnError(errors.New("db"))

	r := NewDecisionRulePostgresRepository(db)
	_, err := r.GetDecisionRuleByScheduleID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecisionRule_GetByScheduleID_DecisionRuleNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	drID := uuid.NewString()
	mock.ExpectQuery(`SELECT .* FROM schedules`).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "DECISION_RULE_ID"}).AddRow(uuid.NewString(), drID))
	// Decision rule lookup returns no rows
	mock.ExpectQuery(`SELECT \* FROM decision_rules`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	r := NewDecisionRulePostgresRepository(db)
	got, err := r.GetDecisionRuleByScheduleID(context.Background(), uuid.New())
	if err != nil || got != nil {
		t.Errorf("DR not found: got=%v err=%v", got, err)
	}
}

func TestDecisionRule_GetByScheduleIDs_NoSchedules(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT .* FROM schedules`).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "DECISION_RULE_ID"}))

	r := NewDecisionRulePostgresRepository(db)
	got, err := r.GetDecisionRuleByScheduleIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err != nil || len(got) != 0 {
		t.Errorf("no schedules: got=%v err=%v", got, err)
	}
}

func TestDecisionRule_GetByScheduleIDs_SchedulesQueryError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT .* FROM schedules`).
		WillReturnError(errors.New("boom"))

	r := NewDecisionRulePostgresRepository(db)
	_, err := r.GetDecisionRuleByScheduleIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected error")
	}
}
