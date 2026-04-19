package main

import (
	"context"
	"embed"
	"fmt"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

//go:embed seeds/*.sql
var embedSeeds embed.FS

func main() {
	// Load .env file if present (ignored in production where env vars are injected)
	err := godotenv.Load()
	if err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "WARN",
			Message: "No .env file found, relying on environment variables",
		})
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: "Starting database migration...",
	})

	// Load PostgreSQL config from environment variables
	cfg := entity.PostgresConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "kbank_ecms"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: "Failed to connect to database: " + err.Error(),
		})
		os.Exit(1)
	}

	// Run auto-migration for all models (creates tables and adds missing columns).
	models := entity.AllModels()
	if err := db.AutoMigrate(models...); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: "AutoMigrate failed: " + err.Error(),
		})
		os.Exit(1)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: fmt.Sprintf("AutoMigrate completed — %d models", len(models)),
	})

	// Sync column constraints (NOT NULL, DEFAULT) from struct tags to existing columns.
	// AutoMigrate only adds new columns; AlterColumn is required to update constraints.
	if err := alterAllModelColumns(db, models); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: "Column constraint sync failed: " + err.Error(),
		})
		os.Exit(1)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: "Column constraints synchronized",
	})

	// Sync indexes (uniqueIndex, index, check, etc.) from struct tags to existing indexes.
	// AutoMigrate only creates missing indexes; it never modifies an existing index
	// (e.g. promoting a plain index to unique). Drop + recreate ensures the DB matches
	// the current struct tag definitions.
	if err := alterAllModelIndexes(db, models); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: "Index sync failed: " + err.Error(),
		})
		os.Exit(1)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: "Indexes synchronized",
	})

	// Sync CHECK constraints from struct tags (gorm:"check:...") to the database.
	// AutoMigrate does not modify existing check constraints.
	if err := alterAllModelConstraints(db, models); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: "Constraint sync failed: " + err.Error(),
		})
		os.Exit(1)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: "Constraints synchronized",
	})

	// Run goose SQL migrations (btree_gist extension + EXCLUDE constraint).
	sqlDB, err := db.DB()
	if err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: fmt.Sprintf("Failed to get sql.DB handle: %s", err.Error()),
		})
		os.Exit(1)
	}

	// Ensure the migration_tracking schema exists so both goose tables land there.
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS migration_tracking").Error; err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: fmt.Sprintf("Failed to create migration_tracking schema: %s", err.Error()),
		})
		os.Exit(1)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: fmt.Sprintf("Failed to set goose dialect: %s", err.Error()),
		})
		os.Exit(1)
	}

	// --- Layer 1: schema migrations (tracked in migration_tracking.goose_migrations) ---
	goose.SetTableName("migration_tracking.goose_migrations")
	goose.SetBaseFS(embedMigrations)

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: fmt.Sprintf("Failed to run goose schema migrations: %s", err.Error()),
		})
		os.Exit(1)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: "Goose schema migrations applied successfully",
	})

	// --- Layer 2: seed migrations (tracked in migration_tracking.seed_migrations) ---
	goose.SetTableName("migration_tracking.seed_migrations")
	goose.SetBaseFS(embedSeeds)

	if err := goose.Up(sqlDB, "seeds"); err != nil {
		logger.LStartup(context.Background(), entity.StartupLog{
			Service: "MIGRATE",
			Level:   "FATAL",
			Message: fmt.Sprintf("Failed to run goose seed migrations: %s", err.Error()),
		})
		os.Exit(1)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "MIGRATE",
		Level:   "INFO",
		Message: "Goose seed migrations applied successfully",
	})
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// alterAllModelConstraints drops and recreates every CHECK constraint defined via
// gorm:"check:..." struct tags so that expression changes are applied to existing
// tables. AutoMigrate only creates missing constraints; it never modifies them.
func alterAllModelConstraints(db *gorm.DB, models []interface{}) error {
	migrator := db.Migrator()
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("parse model %T: %w", model, err)
		}
		for name := range stmt.Schema.ParseCheckConstraints() {
			if migrator.HasConstraint(model, name) {
				if err := migrator.DropConstraint(model, name); err != nil {
					logger.LStartup(context.Background(), entity.StartupLog{
						Service: "MIGRATE",
						Level:   "WARN",
						Message: fmt.Sprintf("DropConstraint %s.%s: %s", stmt.Schema.Table, name, err.Error()),
					})
					continue
				}
			}
			if migrator.HasConstraint(model, name) {
				continue
			}
			if err := migrator.CreateConstraint(model, name); err != nil {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "MIGRATE",
					Level:   "WARN",
					Message: fmt.Sprintf("CreateConstraint %s.%s: %s", stmt.Schema.Table, name, err.Error()),
				})
			}
		}
	}
	return nil
}

// alterAllModelIndexes drops and recreates every index defined in model struct tags so
// that changes (e.g. plain index → uniqueIndex, new composite index) are applied to
// existing tables. AutoMigrate only creates missing indexes; it never modifies them.
func alterAllModelIndexes(db *gorm.DB, models []interface{}) error {
	migrator := db.Migrator()
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("parse model %T: %w", model, err)
		}
		for _, idx := range stmt.Schema.ParseIndexes() {
			// Drop the index if it already exists so the recreation below reflects
			// any definition change (uniqueness, columns, partial condition, etc.).
			if migrator.HasIndex(model, idx.Name) {
				if err := migrator.DropIndex(model, idx.Name); err != nil {
					logger.LStartup(context.Background(), entity.StartupLog{
						Service: "MIGRATE",
						Level:   "WARN",
						Message: fmt.Sprintf("DropIndex %s.%s: %s", stmt.Schema.Table, idx.Name, err.Error()),
					})
					continue
				}
			}
			if migrator.HasIndex(model, idx.Name) {
				continue
			}
			if err := migrator.CreateIndex(model, idx.Name); err != nil {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "MIGRATE",
					Level:   "WARN",
					Message: fmt.Sprintf("CreateIndex %s.%s: %s", stmt.Schema.Table, idx.Name, err.Error()),
				})
			}
		}
	}
	return nil
}

// alterAllModelColumns calls db.Migrator().AlterColumn() for every column of every
// model so that NOT NULL / DEFAULT changes in struct tags are applied to existing
// PostgreSQL columns. AutoMigrate alone does not modify existing column definitions.
func alterAllModelColumns(db *gorm.DB, models []interface{}) error {
	migrator := db.Migrator()
	legacyNamer := schema.NamingStrategy{}
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("parse model %T: %w", model, err)
		}
		for _, field := range stmt.Schema.Fields {
			// Skip fields without a real DB column (associations, ignored fields).
			if field.DBName == "" {
				continue
			}
			// PostgreSQL does not allow ALTER COLUMN on primary key columns (SQLSTATE 42P16).
			if field.PrimaryKey {
				continue
			}

			if err := renameLegacyColumnToUpperSnake(migrator, model, stmt.Schema.Table, field, legacyNamer); err != nil {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "MIGRATE",
					Level:   "WARN",
					Message: fmt.Sprintf("RenameColumn %s.%s: %s", stmt.Schema.Table, field.DBName, err.Error()),
				})
				continue
			}

			if !migrator.HasColumn(model, field.DBName) {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "MIGRATE",
					Level:   "WARN",
					Message: fmt.Sprintf("Skip AlterColumn %s.%s: column does not exist", stmt.Schema.Table, field.DBName),
				})
				continue
			}

			if err := migrator.AlterColumn(model, field.DBName); err != nil {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "MIGRATE",
					Level:   "WARN",
					Message: fmt.Sprintf("AlterColumn %s.%s: %s", stmt.Schema.Table, field.DBName, err.Error()),
				})
			}
		}
	}
	return nil
}

func renameLegacyColumnToUpperSnake(
	migrator gorm.Migrator,
	model interface{},
	table string,
	field *schema.Field,
	legacyNamer schema.NamingStrategy,
) error {
	legacyColumnName := legacyNamer.ColumnName(table, field.Name)
	if legacyColumnName == "" || legacyColumnName == field.DBName || migrator.HasColumn(model, field.DBName) || !migrator.HasColumn(model, legacyColumnName) {
		return nil
	}
	return migrator.RenameColumn(model, legacyColumnName, field.DBName)
}
