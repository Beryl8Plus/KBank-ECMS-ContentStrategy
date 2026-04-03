package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"kbank-ecms/internal/model"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	// ctx = context.Background() // Removed global context
)

// InitRedis initializes the Redis client with TLS support for Azure Redis Enterprise
func InitRedis(ctx context.Context, cfg model.RedisConfig) error {
	SETENV := os.Getenv("SETENV")

	// If DEVLOCAL, use basic setup
	if SETENV == "DEVLOCAL" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Password: cfg.Password,
		})
	} else {
		// Default to ENV variable, if not set use the hardcoded one (as fallback/placeholder)
		principalID := os.Getenv("REDIS_PRINCIPAL_ID")
		redisResourceID := "acca5fbb-b7e4-4009-81f1-37e38fd66d78" // https://learn.microsoft.com/en-us/azure/redis/entra-for-authentication

		opts := &redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Username: principalID,
			TLSConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, // Skip certificate verification for internal IP-based connections
			},
		}

		if cfg.Password != "" {
			opts.Password = cfg.Password
		} else {
			// Use Workload Identity if password is not provided
			cred, err := azidentity.NewWorkloadIdentityCredential(nil)
			if err != nil {
				return fmt.Errorf("failed to create workload identity credential: %w", err)
			}

			opts.CredentialsProvider = func() (string, string) {
					// Get token for Redis
				token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
					Scopes: []string{redisResourceID + "/.default"},
				})
				if err != nil {
					fmt.Printf("failed to get redis token: %v\n", err)
					return principalID, ""
				}
				return principalID, token.Token
			}
		}
		rdb = redis.NewClient(opts)
	}

	// Ping to check connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to redis %s:%s: %w", cfg.Host, cfg.Port, err)
	}

	LStartup(model.StartupLog{
		Service: "REDIS",
		Level:   "INFO",
		Message: "Connected to Azure Redis Enterprise successfully",
	})

	return nil
}

// GetCache retrieves a value from Redis by key
func GetCache(ctx context.Context, key string) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetCache sets a value in Redis (for testing)
func SetCache(ctx context.Context, key string, value string, expiration time.Duration) error {
	if rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return rdb.Set(ctx, key, value, expiration).Err()
}

// GetHashCache retrieves a value from a Redis Hash by key and field
func GetHashCache(ctx context.Context, key string, field string) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	val, err := rdb.HGet(ctx, key, field).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetHashCache sets a value in a Redis Hash (for testing)
func SetHashCache(ctx context.Context, key string, field string, value string) error {
	if rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return rdb.HSet(ctx, key, field, value).Err()
}

// FlushDB flushes the currently selected Redis database.
// It returns an error if the Redis client is not initialized or the command fails.
func FlushDB(ctx context.Context) error {
	if rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return rdb.FlushDB(ctx).Err()
}
