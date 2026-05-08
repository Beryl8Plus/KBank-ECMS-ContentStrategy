package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

func TestScheduleOccurrence_Upsert_EmptySliceNoOp(t *testing.T) {
	db, _ := newMockDB(t)
	r := NewScheduleOccurrencePostgresRepository(db)
	if err := r.UpsertOccurrences(context.Background(), nil); err != nil {
		t.Errorf("nil slice: %v", err)
	}
	if err := r.UpsertOccurrences(context.Background(), []*entity.ScheduleOccurrence{}); err != nil {
		t.Errorf("empty slice: %v", err)
	}
}

func TestScheduleOccurrence_DeleteFutureByScheduleID_Error(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM schedule_occurrences`).
		WillReturnError(errors.New("db"))
	mock.ExpectRollback()

	r := NewScheduleOccurrencePostgresRepository(db)
	err := r.DeleteFutureByScheduleID(context.Background(), uuid.New(), time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleOccurrence_DeleteFutureByScheduleID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM schedule_occurrences`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	r := NewScheduleOccurrencePostgresRepository(db)
	if err := r.DeleteFutureByScheduleID(context.Background(), uuid.New(), time.Now()); err != nil {
		t.Errorf("unexpected: %v", err)
	}
}

func TestScheduleOccurrence_DeletePast_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM schedule_occurrences`).
		WillReturnResult(sqlmock.NewResult(0, 5))
	mock.ExpectCommit()

	r := NewScheduleOccurrencePostgresRepository(db)
	if err := r.DeletePastOccurrences(context.Background(), time.Now()); err != nil {
		t.Errorf("unexpected: %v", err)
	}
}

func TestScheduleOccurrence_DeletePast_Error(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM schedule_occurrences`).
		WillReturnError(errors.New("db"))
	mock.ExpectRollback()

	r := NewScheduleOccurrencePostgresRepository(db)
	if err := r.DeletePastOccurrences(context.Background(), time.Now()); err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleOccurrence_ListByScheduleID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM schedule_occurrences`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM schedule_occurrences`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(uuid.NewString()))

	r := NewScheduleOccurrencePostgresRepository(db)
	got, total, err := r.ListByScheduleID(context.Background(), uuid.New(), 1, 10)
	if err != nil || total != 1 || len(got) != 1 {
		t.Errorf("unexpected: total=%d len=%d err=%v", total, len(got), err)
	}
}

func TestScheduleOccurrence_ListByScheduleID_CountError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM schedule_occurrences`).
		WillReturnError(errors.New("db"))

	r := NewScheduleOccurrencePostgresRepository(db)
	_, _, err := r.ListByScheduleID(context.Background(), uuid.New(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleOccurrence_ListActiveAt_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM schedule_occurrences`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	r := NewScheduleOccurrencePostgresRepository(db)
	got, err := r.ListActiveAt(context.Background(), time.Now())
	if err != nil || got == nil {
		t.Errorf("unexpected: err=%v got=%v", err, got)
	}
}

func TestScheduleOccurrence_ListActiveAt_Error(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM schedule_occurrences`).
		WillReturnError(errors.New("db"))

	r := NewScheduleOccurrencePostgresRepository(db)
	if _, err := r.ListActiveAt(context.Background(), time.Now()); err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleOccurrence_ExpireEnded_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE schedule_occurrences`).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	r := NewScheduleOccurrencePostgresRepository(db)
	got, err := r.ExpireEndedOccurrences(context.Background(), time.Now())
	if err != nil || got != 3 {
		t.Errorf("unexpected: got=%d err=%v", got, err)
	}
}

func TestScheduleOccurrence_ExpireEnded_Error(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE schedule_occurrences`).
		WillReturnError(errors.New("db"))
	mock.ExpectRollback()

	r := NewScheduleOccurrencePostgresRepository(db)
	if _, err := r.ExpireEndedOccurrences(context.Background(), time.Now()); err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleOccurrence_ListActiveByPlacementsAt_EmptyShortCircuits(t *testing.T) {
	db, _ := newMockDB(t)
	r := NewScheduleOccurrencePostgresRepository(db)
	got, err := r.ListActiveByPlacementsAt(context.Background(), nil, time.Now())
	if err != nil || got != nil {
		t.Errorf("empty placements should return (nil, nil), got (%v, %v)", got, err)
	}
}

func TestScheduleOccurrence_ListActiveByPlacementsAt_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT .* FROM schedule_occurrences`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	r := NewScheduleOccurrencePostgresRepository(db)
	got, err := r.ListActiveByPlacementsAt(context.Background(), []string{"hero"}, time.Now())
	if err != nil || got == nil {
		t.Errorf("unexpected: err=%v got=%v", err, got)
	}
}

func TestScheduleOccurrence_ListActiveByPlacementsAt_Error(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT .* FROM schedule_occurrences`).
		WillReturnError(errors.New("db"))

	r := NewScheduleOccurrencePostgresRepository(db)
	if _, err := r.ListActiveByPlacementsAt(context.Background(), []string{"hero"}, time.Now()); err == nil {
		t.Fatal("expected error")
	}
}
