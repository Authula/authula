package secondarystorage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/GoBetterAuth/go-better-auth/env"
)

// RedisSecondaryStorageOptions configures a Redis secondary storage instance
type RedisSecondaryStorageOptions struct {
	URL         string
	MaxRetries  int
	PoolSize    int
	PoolTimeout time.Duration
}

// RedisSecondaryStorage implements SecondaryStorage using Redis as the backend
type RedisSecondaryStorage struct {
	client *redis.Client
}

// NewRedisSecondaryStorage creates a new Redis-backed secondary storage instance
func NewRedisSecondaryStorage(opts RedisSecondaryStorageOptions) (*RedisSecondaryStorage, error) {
	envURL := os.Getenv(env.EnvRedisURL)
	if envURL != "" {
		opts.URL = envURL
	} else {
		if opts.URL == "" {
			panic("redis URL must be provided")
		}
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.PoolSize == 0 {
		opts.PoolSize = 10
	}
	if opts.PoolTimeout == 0 {
		opts.PoolTimeout = 30 * time.Second
	}

	opt, err := redis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	opt.MaxRetries = opts.MaxRetries
	opt.PoolSize = opts.PoolSize
	opt.PoolTimeout = opts.PoolTimeout
	opt.ReadTimeout = 5 * time.Second
	opt.WriteTimeout = 5 * time.Second

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", opts.URL, err)
	}

	return &RedisSecondaryStorage{
		client: client,
	}, nil
}

// Get retrieves a value from Redis by key
// Returns nil if the key does not exist
func (rs *RedisSecondaryStorage) Get(ctx context.Context, key string) (any, error) {
	val, err := rs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}
	return val, nil
}

// Set stores a value in Redis with an optional TTL
// The value must be a string.
// TODO: update this to support other types as needed
func (rs *RedisSecondaryStorage) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be of type string, got %T", value)
	}

	var expiration time.Duration
	if ttl != nil {
		expiration = *ttl
	}

	if err := rs.client.Set(ctx, key, valueStr, expiration).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}
	return nil
}

// Delete removes a key from Redis
func (rs *RedisSecondaryStorage) Delete(ctx context.Context, key string) error {
	if err := rs.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}
	return nil
}

// Incr atomically increments an integer value in Redis by 1
// If the key does not exist, it is set to 1
// If a TTL is provided, it is only applied on key creation
func (rs *RedisSecondaryStorage) Incr(ctx context.Context, key string, ttl *time.Duration) (int, error) {
	exists, err := rs.client.Exists(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis exists check error: %w", err)
	}

	val, err := rs.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr error: %w", err)
	}

	if exists == 0 && ttl != nil {
		if err := rs.client.Expire(ctx, key, *ttl).Err(); err != nil {
			return 0, fmt.Errorf("redis expire error: %w", err)
		}
	}

	return int(val), nil
}

// TTL returns the remaining time to live for a key in seconds
// Returns nil if the key does not exist or has no expiration
func (rs *RedisSecondaryStorage) TTL(ctx context.Context, key string) (*time.Duration, error) {
	ttl, err := rs.client.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ttl error: %w", err)
	}

	// Redis returns -1 if key exists but has no associated expire
	// Redis returns -2 if key does not exist
	if ttl == -1 || ttl == -2 {
		return nil, nil
	}

	return &ttl, nil
}

// Close closes the Redis connection
func (rs *RedisSecondaryStorage) Close() error {
	if rs.client != nil {
		return rs.client.Close()
	}
	return nil
}
