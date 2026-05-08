package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
)

// newMockDB returns a GORM DB backed by go-sqlmock with the same naming
// strategy as production. Tests register expected queries on the mock and
// then exercise repository methods.
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
		WithoutQuotingCheck:  true,
	}), &gorm.Config{
		NamingStrategy: database.UpperSnakeColumnNamingStrategy{},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	t.Cleanup(func() {
		sqlDB.Close()
	})
	return gormDB, mock
}

// ─────────────────────────────────────────────────────────────────────────────
// CLENSchemaRegistry
// ─────────────────────────────────────────────────────────────────────────────

func TestCLENSchemaRegistry_GetByID_NilShortCircuit(t *testing.T) {
	db, _ := newMockDB(t)
	r := NewCLENSchemaRegistryPostgresRepository(db)
	got, err := r.GetByID(context.Background(), uuid.Nil)
	if err != nil || got != nil {
		t.Errorf("nil ID should return (nil, nil), got (%v, %v)", got, err)
	}
}

func TestCLENSchemaRegistry_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM clen_schema_registry`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	r := NewCLENSchemaRegistryPostgresRepository(db)
	got, err := r.GetByID(context.Background(), uuid.New())
	if err != nil || got != nil {
		t.Errorf("not-found should return (nil, nil), got (%v, %v)", got, err)
	}
}

func TestCLENSchemaRegistry_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	id := uuid.New()
	mock.ExpectQuery(`SELECT \* FROM clen_schema_registry`).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SCHEMA_NAME"}).
			AddRow(id.String(), "ARS"))

	r := NewCLENSchemaRegistryPostgresRepository(db)
	got, err := r.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.SchemaName != "ARS" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestCLENSchemaRegistry_GetByID_TransportError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM clen_schema_registry`).
		WillReturnError(sql.ErrConnDone)

	r := NewCLENSchemaRegistryPostgresRepository(db)
	_, err := r.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected transport error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AttributePostgresRepository
// ─────────────────────────────────────────────────────────────────────────────

func TestAttributeRepository_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM attributes`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	r := NewAttributePostgresRepository(db)
	got, err := r.GetAttributeByID(context.Background(), uuid.New())
	if err != nil || got != nil {
		t.Errorf("not-found: got=%v err=%v", got, err)
	}
}

func TestAttributeRepository_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	id := uuid.New()
	mock.ExpectQuery(`SELECT \* FROM attributes`).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "FIELD_NAME"}).
			AddRow(id.String(), "tot_pnt"))

	r := NewAttributePostgresRepository(db)
	got, err := r.GetAttributeByID(context.Background(), id)
	if err != nil || got == nil || got.FieldName != "tot_pnt" {
		t.Errorf("got=%+v err=%v", got, err)
	}
}

func TestAttributeRepository_GetByID_TransportError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM attributes`).
		WillReturnError(errors.New("db down"))

	r := NewAttributePostgresRepository(db)
	_, err := r.GetAttributeByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAttributeRepository_ListByTableSourceName_EmptyShortCircuits(t *testing.T) {
	db, _ := newMockDB(t)
	r := NewAttributePostgresRepository(db)
	got, err := r.ListByTableSourceName(context.Background(), "")
	if err != nil || got != nil {
		t.Errorf("empty datasource should return (nil,nil), got (%v, %v)", got, err)
	}
}

func TestAttributeRepository_ListByTableSourceName_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM attributes`).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "FIELD_NAME"}).
			AddRow(uuid.NewString(), "a").
			AddRow(uuid.NewString(), "b"))

	r := NewAttributePostgresRepository(db)
	got, err := r.ListByTableSourceName(context.Background(), "ds")
	if err != nil || len(got) != 2 {
		t.Errorf("got len=%d err=%v", len(got), err)
	}
}

func TestAttributeRepository_ListPaginated_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM attributes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT \* FROM attributes`).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "FIELD_NAME"}).
			AddRow(uuid.NewString(), "a").
			AddRow(uuid.NewString(), "b"))

	r := NewAttributePostgresRepository(db)
	list, total, err := r.ListAttributesPaginated(context.Background(), 1, 10)
	if err != nil || total != 2 || len(list) != 2 {
		t.Errorf("unexpected: total=%d len=%d err=%v", total, len(list), err)
	}
}

func TestAttributeRepository_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO attributes`).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(uuid.NewString()))
	mock.ExpectCommit()

	r := NewAttributePostgresRepository(db)
	a := &entity.Attribute{
		BaseModel: entity.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		FieldName: "x",
	}
	if err := r.CreateAttribute(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestAttributeRepository_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE attributes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	r := NewAttributePostgresRepository(db)
	a := &entity.Attribute{
		BaseModel: entity.BaseModel{ID: uuid.New(), UpdatedAt: time.Now()},
		FieldName: "x",
	}
	if err := r.UpdateAttribute(context.Background(), a); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestAttributeRepository_Delete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE attributes SET DELETED_AT`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	r := NewAttributePostgresRepository(db)
	if err := r.DeleteAttribute(context.Background(), uuid.New()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
