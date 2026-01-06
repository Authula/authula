package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// SecondaryStorageProvider wraps a SecondaryStorage backend for rate limiting
// This allows rate limits to use distributed storage (Redis, database) instead of in-memory
type SecondaryStorageProvider struct {
	name    string
	storage models.SecondaryStorage
}

// NewSecondaryStorageProvider creates a new provider wrapping a SecondaryStorage backend
func NewSecondaryStorageProvider(name string, storage models.SecondaryStorage) *SecondaryStorageProvider {
	return &SecondaryStorageProvider{
		name:    name,
		storage: storage,
	}
}

// GetName returns the provider name
func (p *SecondaryStorageProvider) GetName() string {
	return p.name
}

// CheckAndIncrement checks if a request is allowed and increments the counter
func (p *SecondaryStorageProvider) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	select {
	case <-ctx.Done():
		return false, 0, time.Time{}, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	now := time.Now()

	existing, err := p.storage.Get(ctx, key)
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to get rate limit count: %w", err)
	}

	var resetTime time.Time
	if existing == nil {
		// New entry - increment with TTL set to window
		count, err := p.storage.Incr(ctx, key, &window)
		if err != nil {
			return false, 0, time.Time{}, fmt.Errorf("failed to increment rate limit: %w", err)
		}
		resetTime = now.Add(window)
		return count <= maxRequests, count, resetTime, nil
	}

	// Existing entry - increment without updating TTL
	count, err := p.storage.Incr(ctx, key, nil)
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	// Get the remaining TTL from storage
	ttl, err := p.storage.TTL(ctx, key)
	if err != nil || ttl == nil || *ttl <= 0 {
		// Fallback if TTL can't be retrieved
		resetTime = now.Add(window)
	} else {
		// Calculate reset time based on actual TTL remaining
		resetTime = now.Add(*ttl)
	}

	return count <= maxRequests, count, resetTime, nil
}

// Close closes the provider (no-op since we don't own the storage)
func (p *SecondaryStorageProvider) Close() error {
	return nil
}
