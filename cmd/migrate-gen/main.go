package main

import (
	"bytes"
	"context"
	"fmt"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	serviceName = "MIGRATE-GEN"

	// Directory where production goose migration files are written.
	prodMigrationsDir = "cmd/migrate-gen/migrations-prod"

	// File that stores the last-known schema for incremental diffing.
	snapshotFile = "cmd/migrate-gen/migrations-prod/schema_snapshot.sql"
)

// Atlas exclude patterns (tracking tables, goose metadata).
var atlasExcludes = []string{
	"migration_tracking.*",
	"goose_migrations",
	"seed_migrations",
	"mock_migrations",
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	// Load .env file if present (ignored in CI where env vars are injected).
	_ = godotenv.Load()

	logInfo("Starting migration generation pipeline...")

	cfg := config.PostgresConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "kbank_ecms"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	// ── Step 1: Connect to the shadow (empty) Postgres ────────────────
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		logFatal("Failed to connect to shadow database: " + err.Error())
	}
	logInfo("Connected to shadow database")

	// ── Step 2: Run GORM AutoMigrate to build the desired schema ──────
	models := entity.AllModels()
	if err := db.AutoMigrate(models...); err != nil {
		logFatal("AutoMigrate failed: " + err.Error())
	}
	logInfo(fmt.Sprintf("AutoMigrate completed — %d models", len(models)))

	// Sync column constraints (NOT NULL, DEFAULT).
	if err := alterAllModelColumns(db, models); err != nil {
		logFatal("Column constraint sync failed: " + err.Error())
	}
	logInfo("Column constraints synchronized")

	// Sync indexes.
	if err := alterAllModelIndexes(db, models); err != nil {
		logFatal("Index sync failed: " + err.Error())
	}
	logInfo("Indexes synchronized")

	// Sync CHECK constraints.
	if err := alterAllModelConstraints(db, models); err != nil {
		logFatal("Constraint sync failed: " + err.Error())
	}
	logInfo("Constraints synchronized")

	// ── Step 3: Run existing goose migrations on the shadow DB ────────
	sqlDB, err := db.DB()
	if err != nil {
		logFatal("Failed to get sql.DB handle: " + err.Error())
	}

	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS migration_tracking").Error; err != nil {
		logFatal("Failed to create migration_tracking schema: " + err.Error())
	}

	if err := goose.SetDialect("postgres"); err != nil {
		logFatal("Failed to set goose dialect: " + err.Error())
	}

	goose.SetTableName("migration_tracking.goose_migrations")

	// Use os.DirFS to load migration files at runtime (go:embed doesn't support ".." paths).
	migrationsFS := os.DirFS("cmd/migrate/migrations")
	goose.SetBaseFS(migrationsFS)

	if err := goose.Up(sqlDB, "."); err != nil {
		logFatal("Failed to run goose schema migrations: " + err.Error())
	}
	logInfo("Goose dev migrations applied to shadow DB")

	// ── Step 4: Capture the current (desired) schema via Atlas inspect ─
	atlasDBURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=public",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	currentSchema, err := atlasSchemaInspect(atlasDBURL)
	if err != nil {
		logFatal("Atlas schema inspect failed: " + err.Error())
	}
	if strings.TrimSpace(currentSchema) == "" {
		logFatal("Atlas schema inspect returned empty output — check Atlas installation and DB connection")
	}
	logInfo(fmt.Sprintf("Current schema captured (%d bytes)", len(currentSchema)))

	// ── Step 5: Read the previous schema snapshot ─────────────────────
	previousSchema := ""
	if data, err := os.ReadFile(snapshotFile); err == nil {
		previousSchema = string(data)
		logInfo(fmt.Sprintf("Previous snapshot loaded (%d bytes)", len(previousSchema)))
	} else {
		logInfo("No previous snapshot found — treating as initial generation")
	}

	// ── Step 6: Compare schemas ───────────────────────────────────────
	if normalizeSQL(currentSchema) == normalizeSQL(previousSchema) {
		logInfo("No schema changes detected — skipping migration generation")
		os.Exit(0)
	}
	logInfo("Schema changes detected!")

	// ── Step 7: Compute incremental diff ──────────────────────────────
	var upSQL, downSQL string

	if previousSchema == "" {
		// First-ever generation: compute UP from empty → current using Atlas.
		upSQL, err = computeDiffFromHCL("", currentSchema, cfg)
		if err != nil {
			logWarn("Atlas HCL diff failed for initial generation, using inspect SQL directly: " + err.Error())
			upSQL, err = computeDiffFromSQL("", currentSchema, cfg)
			if err != nil {
				logFatal("Failed to compute initial diff: " + err.Error())
			}
		}
		downSQL = generateDropAllFromHCL(currentSchema)
	} else {
		// Incremental diff: previous → current.
		upSQL, err = computeDiffFromHCL(previousSchema, currentSchema, cfg)
		if err != nil {
			logWarn("Atlas HCL diff failed, trying SQL-based diff: " + err.Error())
			upSQL, err = computeDiffFromSQL(previousSchema, currentSchema, cfg)
			if err != nil {
				logFatal("Failed to compute incremental diff: " + err.Error())
			}
		}
		// Reverse diff: current → previous for the DOWN migration.
		downSQL, err = computeDiffFromHCL(currentSchema, previousSchema, cfg)
		if err != nil {
			logWarn("Atlas HCL reverse diff failed, trying SQL-based diff: " + err.Error())
			downSQL, err = computeDiffFromSQL(currentSchema, previousSchema, cfg)
			if err != nil {
				logWarn("SQL-based reverse diff also failed: " + err.Error())
				downSQL = "-- Unable to auto-generate DOWN migration; please write manually."
			}
		}
	}

	if strings.TrimSpace(upSQL) == "" {
		logInfo("Diff produced empty output — no migration needed")
		os.Exit(0)
	}

	logInfo(fmt.Sprintf("UP migration SQL (%d bytes):\n%s", len(upSQL), upSQL))

	// ── Step 8: Write goose migration file ────────────────────────────
	timestamp := time.Now().Format("20060102150405")
	migrationFilename := fmt.Sprintf("%s_auto_generated.sql", timestamp)
	migrationPath := filepath.Join(prodMigrationsDir, migrationFilename)

	gooseContent := formatGooseMigration(upSQL, downSQL)

	if err := os.MkdirAll(prodMigrationsDir, 0o755); err != nil {
		logFatal("Failed to create migrations-prod directory: " + err.Error())
	}

	if err := os.WriteFile(migrationPath, []byte(gooseContent), 0o644); err != nil {
		logFatal("Failed to write migration file: " + err.Error())
	}
	logInfo(fmt.Sprintf("✅ Migration file written: %s", migrationPath))

	// ── Step 9: Update schema snapshot ────────────────────────────────
	if err := os.WriteFile(snapshotFile, []byte(currentSchema), 0o644); err != nil {
		logFatal("Failed to update schema snapshot: " + err.Error())
	}
	logInfo("Schema snapshot updated")

	logInfo("Migration generation pipeline completed successfully!")
}

// ---------------------------------------------------------------------------
// Atlas helpers
// ---------------------------------------------------------------------------

// atlasSchemaInspect captures the current database schema in Atlas HCL format.
// This is deterministic and produces consistent output for the same schema state.
func atlasSchemaInspect(dbURL string) (string, error) {
	args := []string{
		"schema", "inspect",
		"-u", dbURL,
	}
	for _, pattern := range atlasExcludes {
		args = append(args, "--exclude", pattern)
	}

	logInfo(fmt.Sprintf("Running: atlas %s", strings.Join(args, " ")))

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("atlas", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("atlas schema inspect failed: %w\nstderr: %s", err, stderr.String())
	}

	result := stdout.String()
	if stderr.Len() > 0 {
		logWarn("Atlas stderr: " + stderr.String())
	}

	return result, nil
}

// computeDiffFromHCL computes a SQL diff between two Atlas HCL schemas using
// a dev database for normalization. Returns the SQL migration statements.
func computeDiffFromHCL(fromHCL, toHCL string, cfg config.PostgresConfig) (string, error) {
	tmpDir, err := os.MkdirTemp("", "migrate-gen-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write schemas to temp files.
	fromFile := filepath.Join(tmpDir, "from.hcl")
	toFile := filepath.Join(tmpDir, "to.hcl")

	fromContent := fromHCL
	if fromContent == "" {
		// Empty schema = no tables.
		fromContent = "// empty schema\n"
	}

	if err := os.WriteFile(fromFile, []byte(fromContent), 0o644); err != nil {
		return "", fmt.Errorf("write from schema: %w", err)
	}
	if err := os.WriteFile(toFile, []byte(toHCL), 0o644); err != nil {
		return "", fmt.Errorf("write to schema: %w", err)
	}

	// Create a dev database for Atlas normalization.
	devDBName := cfg.DBName + "_atlas_dev"
	ensureDevDB(cfg, devDBName)
	defer dropDB(cfg, devDBName)

	devURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, devDBName, cfg.SSLMode,
	)

	args := []string{
		"schema", "diff",
		"--from", "file://" + fromFile,
		"--to", "file://" + toFile,
		"--dev-url", devURL,
		"--format", `{{ sql . "  " }}`,
	}

	logInfo(fmt.Sprintf("Running: atlas schema diff (from=%d bytes, to=%d bytes)", len(fromContent), len(toHCL)))

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("atlas", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("atlas schema diff: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// computeDiffFromSQL is a fallback that applies HCL schemas to temporary
// databases via `atlas schema apply`, then diffs the live databases.
func computeDiffFromSQL(fromHCL, toHCL string, cfg config.PostgresConfig) (string, error) {
	// Create two temporary databases.
	fromDBName := cfg.DBName + "_diff_from"
	toDBName := cfg.DBName + "_diff_to"

	ensureDevDB(cfg, fromDBName)
	ensureDevDB(cfg, toDBName)
	defer dropDB(cfg, fromDBName)
	defer dropDB(cfg, toDBName)

	// Apply HCL schemas to their respective databases via atlas schema apply.
	if fromHCL != "" {
		if err := applyHCLToDB(cfg, fromDBName, fromHCL); err != nil {
			return "", fmt.Errorf("apply from schema: %w", err)
		}
	}
	if toHCL != "" {
		if err := applyHCLToDB(cfg, toDBName, toHCL); err != nil {
			return "", fmt.Errorf("apply to schema: %w", err)
		}
	}

	fromURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, fromDBName, cfg.SSLMode,
	)
	toURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, toDBName, cfg.SSLMode,
	)

	args := []string{
		"schema", "diff",
		"--from", fromURL,
		"--to", toURL,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("atlas", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("atlas schema diff (DB): %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// applyHCLToDB applies an Atlas HCL schema to a database using `atlas schema apply`.
func applyHCLToDB(cfg config.PostgresConfig, dbName, hcl string) error {
	tmpDir, err := os.MkdirTemp("", "migrate-gen-apply-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	hclFile := filepath.Join(tmpDir, "schema.hcl")
	if err := os.WriteFile(hclFile, []byte(hcl), 0o644); err != nil {
		return fmt.Errorf("write HCL file: %w", err)
	}

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, dbName, cfg.SSLMode,
	)

	args := []string{
		"schema", "apply",
		"-u", dbURL,
		"--to", "file://" + hclFile,
		"--auto-approve",
	}

	var stderr bytes.Buffer
	cmd := exec.Command("atlas", args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("atlas schema apply: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// ensureDevDB creates a fresh database and installs required extensions.
func ensureDevDB(cfg config.PostgresConfig, dbName string) {
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		logWarn(fmt.Sprintf("Could not connect to create dev DB %s: %s", dbName, err.Error()))
		return
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Terminate existing connections before dropping.
	db.Exec(fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
		dbName,
	))
	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	logInfo(fmt.Sprintf("Created dev database: %s", dbName))

	// Install btree_gist extension (required for EXCLUDE USING GIST on uuid columns).
	devDB, err := database.NewPostgresDB(config.PostgresConfig{
		Host: cfg.Host, Port: cfg.Port, User: cfg.User,
		Password: cfg.Password, DBName: dbName, SSLMode: cfg.SSLMode,
	})
	if err != nil {
		logWarn(fmt.Sprintf("Could not connect to dev DB %s to install extensions: %s", dbName, err.Error()))
		return
	}
	devSqlDB, _ := devDB.DB()
	defer devSqlDB.Close()

	devDB.Exec("CREATE EXTENSION IF NOT EXISTS btree_gist")
}

// dropDB drops a database.
func dropDB(cfg config.PostgresConfig, dbName string) {
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		return
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	db.Exec(fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
		dbName,
	))
	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
}

// ---------------------------------------------------------------------------
// Goose formatting
// ---------------------------------------------------------------------------

// formatGooseMigration wraps UP and DOWN SQL in goose migration markers.
func formatGooseMigration(upSQL, downSQL string) string {
	now := time.Now().Format("2006-01-02T15:04:05Z07:00")

	var sb strings.Builder
	sb.WriteString("-- +goose Up\n")
	sb.WriteString(fmt.Sprintf("-- Auto-generated from GORM entity changes on %s\n", now))
	sb.WriteString("-- Source: atlas schema diff (previous snapshot → current GORM state)\n\n")
	sb.WriteString(strings.TrimSpace(upSQL))
	sb.WriteString("\n\n")
	sb.WriteString("-- +goose Down\n")
	sb.WriteString("-- Rollback for auto-generated migration\n\n")
	sb.WriteString(strings.TrimSpace(downSQL))
	sb.WriteString("\n")

	return sb.String()
}

// generateDropAllFromHCL generates DROP statements from an Atlas HCL schema.
func generateDropAllFromHCL(hcl string) string {
	var tables []string
	for _, line := range strings.Split(hcl, "\n") {
		trimmed := strings.TrimSpace(line)
		// Atlas HCL format: table "table_name" { ...
		if strings.HasPrefix(trimmed, "table ") && strings.Contains(trimmed, "{") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				tableName := strings.Trim(parts[1], `"`)
				tables = append(tables, fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE;`, tableName))
			}
		}
	}

	// Reverse for dependency order.
	for i, j := 0, len(tables)-1; i < j; i, j = i+1, j-1 {
		tables[i], tables[j] = tables[j], tables[i]
	}

	if len(tables) == 0 {
		return "-- Unable to auto-generate DOWN migration; please write manually."
	}

	return strings.Join(tables, "\n")
}

// ---------------------------------------------------------------------------
// Normalization
// ---------------------------------------------------------------------------

// normalizeSQL normalizes schema text for comparison by stripping comments,
// collapsing whitespace, and sorting lines. This avoids false positives from
// non-deterministic Atlas output ordering.
func normalizeSQL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var lines []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and single-line comments for comparison.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lines = append(lines, trimmed)
	}

	// Sort lines so non-deterministic ordering doesn't cause false negatives.
	// This is safe because HCL blocks are self-contained.
	sortedLines := make([]string, len(lines))
	copy(sortedLines, lines)
	// Simple sort — good enough for detecting structural changes.
	for i := 0; i < len(sortedLines); i++ {
		for j := i + 1; j < len(sortedLines); j++ {
			if sortedLines[i] > sortedLines[j] {
				sortedLines[i], sortedLines[j] = sortedLines[j], sortedLines[i]
			}
		}
	}

	return strings.Join(sortedLines, "\n")
}

// ---------------------------------------------------------------------------
// GORM helpers — reused from cmd/migrate/main.go
// ---------------------------------------------------------------------------

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
					logWarn(fmt.Sprintf("DropConstraint %s.%s: %s", stmt.Schema.Table, name, err.Error()))
					continue
				}
			}
			if migrator.HasConstraint(model, name) {
				continue
			}
			if err := migrator.CreateConstraint(model, name); err != nil {
				logWarn(fmt.Sprintf("CreateConstraint %s.%s: %s", stmt.Schema.Table, name, err.Error()))
			}
		}
	}
	return nil
}

func alterAllModelIndexes(db *gorm.DB, models []interface{}) error {
	migrator := db.Migrator()
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("parse model %T: %w", model, err)
		}
		for _, idx := range stmt.Schema.ParseIndexes() {
			if migrator.HasIndex(model, idx.Name) {
				if err := migrator.DropIndex(model, idx.Name); err != nil {
					logWarn(fmt.Sprintf("DropIndex %s.%s: %s", stmt.Schema.Table, idx.Name, err.Error()))
					continue
				}
			}
			if migrator.HasIndex(model, idx.Name) {
				continue
			}
			if err := migrator.CreateIndex(model, idx.Name); err != nil {
				logWarn(fmt.Sprintf("CreateIndex %s.%s: %s", stmt.Schema.Table, idx.Name, err.Error()))
			}
		}
	}
	return nil
}

func alterAllModelColumns(db *gorm.DB, models []interface{}) error {
	migrator := db.Migrator()
	legacyNamer := schema.NamingStrategy{}
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("parse model %T: %w", model, err)
		}
		for _, field := range stmt.Schema.Fields {
			if field.DBName == "" {
				continue
			}
			if field.PrimaryKey {
				continue
			}
			if err := renameLegacyColumnToUpperSnake(migrator, model, stmt.Schema.Table, field, legacyNamer); err != nil {
				logWarn(fmt.Sprintf("RenameColumn %s.%s: %s", stmt.Schema.Table, field.DBName, err.Error()))
				continue
			}
			if !migrator.HasColumn(model, field.DBName) {
				logWarn(fmt.Sprintf("Skip AlterColumn %s.%s: column does not exist", stmt.Schema.Table, field.DBName))
				continue
			}
			if err := migrator.AlterColumn(model, field.DBName); err != nil {
				logWarn(fmt.Sprintf("AlterColumn %s.%s: %s", stmt.Schema.Table, field.DBName, err.Error()))
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

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func logInfo(msg string) {
	logger.LStartup(context.Background(), entity.StartupLog{
		Service: serviceName,
		Level:   "INFO",
		Message: msg,
	})
}

func logWarn(msg string) {
	logger.LStartup(context.Background(), entity.StartupLog{
		Service: serviceName,
		Level:   "WARN",
		Message: msg,
	})
}

func logFatal(msg string) {
	logger.LStartup(context.Background(), entity.StartupLog{
		Service: serviceName,
		Level:   "FATAL",
		Message: msg,
	})
	os.Exit(1)
}
