// Package healthcheck provides reusable health check components for Gin services.
// It wraps github.com/tavsec/gin-healthcheck with project-specific checkers for
// PostgreSQL (via GORM) and Redis, and exposes a single Register function that
// wires everything into a Gin engine at /healthz.
package healthcheck

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	"github.com/tavsec/gin-healthcheck/config"
	"gorm.io/gorm"
)

// ---------- PostgreSQL checker ----------

// PostgresCheck verifies the database connection pool is healthy by executing
// a lightweight "SELECT 1" query with a short timeout.
type PostgresCheck struct {
	db *gorm.DB
}

// NewPostgresCheck creates a new PostgresCheck from a GORM DB instance.
func NewPostgresCheck(db *gorm.DB) *PostgresCheck {
	return &PostgresCheck{db: db}
}

// Pass returns true if the database responds to a simple query within 3 seconds.
func (c *PostgresCheck) Pass() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sqlDB, err := c.db.WithContext(ctx).DB()
	if err != nil {
		return false
	}
	return sqlDB.PingContext(ctx) == nil
}

// Name returns the human-readable checker name used in error reports.
func (c *PostgresCheck) Name() string {
	return "postgres"
}

// ---------- Redis checker ----------

// RedisCheck verifies Redis connectivity by sending a PING command.
type RedisCheck struct {
	client *redis.Client
}

// NewRedisCheck creates a new RedisCheck from a go-redis client.
func NewRedisCheck(client *redis.Client) *RedisCheck {
	return &RedisCheck{client: client}
}

// Pass returns true if Redis responds to PING within 3 seconds.
func (c *RedisCheck) Pass() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Ping(ctx).Err() == nil
}

// Name returns the human-readable checker name used in error reports.
func (c *RedisCheck) Name() string {
	return "redis"
}

// ---------- Registration ----------

// Register adds a /healthz endpoint to the Gin engine with the appropriate
// checkers based on which dependencies are non-nil.
//
// - db (required): GORM database connection
// - redisClient (optional): if nil, Redis check is skipped
func Register(r *gin.Engine, db *gorm.DB, redisClient *redis.Client) {
	var checkers []checks.Check

	if db != nil {
		checkers = append(checkers, NewPostgresCheck(db))
	}

	if redisClient != nil {
		checkers = append(checkers, NewRedisCheck(redisClient))
	}

	_ = healthcheck.New(r, config.DefaultConfig(), checkers)
}
