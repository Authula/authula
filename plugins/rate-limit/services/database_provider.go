package services

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/repositories"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

// DatabaseProvider is a database-backed rate limit provider for persistent rate limiting
type DatabaseProvider struct {
	logger          models.Logger
	db              bun.IDB
	repository      repositories.RateLimitRepository
	cleanupInterval time.Duration
}

// NewDatabaseProvider creates a new database rate limit provider
func NewDatabaseProvider(db bun.IDB) (*DatabaseProvider, error) {
	return NewDatabaseProviderWithConfig(db, types.DatabaseStorageConfig{})
}

// NewDatabaseProviderWithConfig creates a new database rate limit provider with custom config
func NewDatabaseProviderWithConfig(db bun.IDB, config types.DatabaseStorageConfig) (*DatabaseProvider, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute
	}

	provider := &DatabaseProvider{
		db:              db,
		repository:      repositories.NewRateLimitRepository(db),
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

// SetRule stores a per-key rate-limit rule without consuming quota.
func (p *DatabaseProvider) SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error {
	return p.repository.SetRule(ctx, key, int(window.Seconds()), maxRequests)
}

// GetRule retrieves a stored per-key rule. Returns 0, 0, false, nil when not found.
func (p *DatabaseProvider) GetRule(ctx context.Context, key string) (time.Duration, int, bool, error) {
	record, err := p.repository.GetRule(ctx, key)
	if err != nil {
		return 0, 0, false, err
	}
	if record == nil {
		return 0, 0, false, nil
	}
	return time.Duration(record.WindowSeconds) * time.Second, record.MaxRequests, true, nil
}

// DeleteRule removes the stored rule for a key.
func (p *DatabaseProvider) DeleteRule(ctx context.Context, key string) error {
	return p.repository.DeleteRule(ctx, key)
}

// cleanupExpired periodically removes expired rate limit records from the database
func (p *DatabaseProvider) cleanupExpired() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := p.repository.CleanupExpired(context.Background(), time.Now()); err != nil {
			// Log error if available, but don't crash the goroutine
			p.logger.Error("failed to cleanup expired rate limit records", "error", err)
			// This is a best-effort cleanup; failures won't block the rate limiter
		}
	}
}
