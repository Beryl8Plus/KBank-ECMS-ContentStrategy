package repository

import (
	"context"
	"crypto/tls"
	"fmt"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/redis/go-redis/v9"
)

// RedisRepository implements domain repository.CacheRepository using Redis.
type RedisRepository struct {
	client *redis.Client
}

// Compile-time interface check.
var _ domainrepo.RedisCacheRepository = (*RedisRepository)(nil)

// Client returns the underlying Redis client for health checks and diagnostics.
func (r *RedisRepository) Client() *redis.Client {
	return r.client
}

// NewRedisRepository creates a Redis client and returns a RedisRepository.
func NewRedisRepository(ctx context.Context, cfg entity.RedisConfig) (*RedisRepository, error) {
	SETENV := os.Getenv("SETENV")

	var rdb *redis.Client

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
				// InsecureSkipVerify: false, //nolint:gosec // Azure Redis Enterprise presents a valid TLS certificate; validation is required
			},
		}

		if cfg.Password != "" {
			opts.Password = cfg.Password
		} else {
			// Use Workload Identity if password is not provided
			cred, err := azidentity.NewWorkloadIdentityCredential(nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create workload identity credential: %w", err)
			}

			opts.CredentialsProvider = func() (string, string) {
				// Get token for Redis
				token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
					Scopes: []string{redisResourceID + "/.default"},
				})
				if err != nil {
					logger.LSystem(ctx, entity.SystemLog{
						Service: "REDIS",
						Level:   "ERROR",
						Message: fmt.Sprintf("failed to get redis token: %v", err),
					})
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
		return nil, fmt.Errorf("failed to connect to redis %s:%s: %w", cfg.Host, cfg.Port, err)
	}

	logger.LStartup(ctx, entity.StartupLog{
		Service: "REDIS",
		Level:   "INFO",
		Message: fmt.Sprintf("Connected to %s %s:%s successfully",
			func() string {
				if SETENV == "DEVLOCAL" {
					return "Local Redis"
				} else {
					return "Azure Redis Enterprise"
				}
			}(), cfg.Host, cfg.Port),
	})

	return &RedisRepository{client: rdb}, nil
}

// Get retrieves a value from Redis by key.
func (r *RedisRepository) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// Set stores a value in Redis with an expiration.
func (r *RedisRepository) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// HGet retrieves a value from a Redis Hash by key and field.
func (r *RedisRepository) HGet(ctx context.Context, key string, field string) (string, error) {
	val, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// HSet sets a value in a Redis Hash.
func (r *RedisRepository) HSet(ctx context.Context, key string, field string, value string) error {
	return r.client.HSet(ctx, key, field, value).Err()
}

// FlushDB flushes the currently selected Redis database.
func (r *RedisRepository) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// GetSet implements the cache-aside pattern.
// It attempts to fetch the value for key from Redis.
// On a cache miss (redis.Nil), it calls loader, stores the result with the
// given expiration, and returns it. Any other Get or Set error is returned
// directly to the caller.
func (r *RedisRepository) GetSet(ctx context.Context, key string, expiration time.Duration, loader func(ctx context.Context) (string, error)) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == nil {
		return val, nil
	}
	if err != redis.Nil {
		return "", fmt.Errorf("cache get %q: %w", key, err)
	}

	// Cache miss — invoke loader.
	val, err = loader(ctx)
	if err != nil {
		return "", err
	}

	if setErr := r.client.Set(ctx, key, val, expiration).Err(); setErr != nil {
		return "", fmt.Errorf("cache set %q: %w", key, setErr)
	}

	return val, nil
}

// Delete removes the key from Redis. Returns nil if the key does not exist.
func (r *RedisRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Publish broadcasts payload to subscribers of the given Pub/Sub channel.
func (r *RedisRepository) Publish(ctx context.Context, channel string, payload string) error {
	if err := r.client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("publish to %q: %w", channel, err)
	}
	return nil
}

// Subscribe listens to a Redis channel and returns a channel for messages.
func (r *RedisRepository) Subscribe(ctx context.Context, channel string) (<-chan string, error) {
	pubsub := r.client.Subscribe(ctx, channel)

	// Wait for confirmation that subscription is created
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return nil, fmt.Errorf("subscribe to %q: %w", channel, err)
	}

	// Use a buffered channel to prevent blocking the Redis client background goroutine
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		defer pubsub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-pubsub.Channel():
				if !ok {
					// go-redis handles reconnection and resubscription internally.
					// If ok is false, it means the pubsub is closed or the context is done.
					return
				}
				// Non-blocking send to avoid stalling the goroutine if the consumer is slow
				select {
				case ch <- msg.Payload:
				default:
					logger.LSystem(ctx, entity.SystemLog{
						Service: "REDIS",
						Level:   "WARN",
						Message: fmt.Sprintf("pubsub: message dropped on channel %q — consumer buffer full", channel),
					})
				}
			}
		}
	}()

	return ch, nil
}
