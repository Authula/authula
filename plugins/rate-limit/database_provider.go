package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// DatabaseProvider is a database-backed rate limit provider for persistent rate limiting
type DatabaseProvider struct {
	db              bun.IDB
	repository      RateLimitRepository
	cleanupInterval time.Duration
}

// NewDatabaseProvider creates a new database rate limit provider
func NewDatabaseProvider(db bun.IDB) (*DatabaseProvider, error) {
	return NewDatabaseProviderWithConfig(db, DatabaseStorageConfig{})
}

// NewDatabaseProviderWithConfig creates a new database rate limit provider with custom config
func NewDatabaseProviderWithConfig(db bun.IDB, config DatabaseStorageConfig) (*DatabaseProvider, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute
	}

	provider := &DatabaseProvider{
		db:              db,
		repository:      NewRateLimitRepository(db),
		cleanupInterval: cleanupInterval,
	}

	go provider.cleanupExpired()

	return provider, nil
}

// GetName returns the provider name
func (p *DatabaseProvider) GetName() string {
	return "database"
}

// CheckAndIncrement checks if a request is allowed and increments the counter
func (p *DatabaseProvider) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	select {
	case <-ctx.Done():
		return false, 0, time.Time{}, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	record, err := p.repository.UpdateOrCreate(ctx, key, window)
	if err != nil {
		return false, 0, time.Time{}, err
	}

	allowed := record.Count <= maxRequests
	return allowed, record.Count, record.ExpiresAt, nil
}

// Close closes the provider (no-op since we don't own the database connection)
func (p *DatabaseProvider) Close() error {
	return nil
}

// cleanupExpired periodically removes expired rate limit records from the database
func (p *DatabaseProvider) cleanupExpired() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := p.repository.CleanupExpired(context.Background(), time.Now()); err != nil {
			// Log error if available, but don't crash the goroutine
			// This is a best-effort cleanup; failures won't block the rate limiter
		}
	}
}
