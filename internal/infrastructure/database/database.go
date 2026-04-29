package database

import (
	"context"
	"fmt"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// NewPostgresDB creates and returns a new GORM DB connection for PostgreSQL.
func NewPostgresDB(cfg config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		NamingStrategy: UpperSnakeColumnNamingStrategy{
			NamingStrategy: schema.NamingStrategy{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "POSTGRES",
		Level:   "INFO",
		Message: fmt.Sprintf("Connected to PostgreSQL (%s:%s/%s)", cfg.Host, cfg.Port, cfg.DBName),
	})

	return db, nil
}

// NewMigrationDB opens a GORM connection with FK constraint creation disabled.
// Used exclusively for the table/column phase of AutoMigrate so that GORM's
// ReorderModels ordering issues with bidirectional associations (has-many ↔
// belongs-to cycles) cannot cause "relation does not exist" errors.
func NewMigrationDB(cfg config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		NamingStrategy: UpperSnakeColumnNamingStrategy{
			NamingStrategy: schema.NamingStrategy{},
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open migration DB: %w", err)
	}
	return db, nil
}
