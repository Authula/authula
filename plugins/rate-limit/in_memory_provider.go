package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryProvider is a thread-safe in-memory rate limit provider
type InMemoryProvider struct {
	mu              sync.RWMutex
	store           map[string]*inMemoryEntry
	cleanupInterval time.Duration
}

type inMemoryEntry struct {
	count     int
	expiresAt time.Time
}

// NewInMemoryProvider creates a new in-memory rate limit provider
func NewInMemoryProvider() *InMemoryProvider {
	return NewInMemoryProviderWithConfig(MemoryStorageConfig{})
}

// NewInMemoryProviderWithConfig creates a new in-memory rate limit provider with custom config
func NewInMemoryProviderWithConfig(config MemoryStorageConfig) *InMemoryProvider {
	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute
	}

	provider := &InMemoryProvider{
		store:           make(map[string]*inMemoryEntry),
		cleanupInterval: cleanupInterval,
	}

	// Start cleanup goroutine to remove expired entries
	go provider.cleanupExpired()

	return provider
}

// GetName returns the provider name
func (p *InMemoryProvider) GetName() string {
	return string(RateLimitProviderInMemory)
}

// CheckAndIncrement checks if a request is allowed and increments the counter
func (p *InMemoryProvider) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	select {
	case <-ctx.Done():
		return false, 0, time.Time{}, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	entry, exists := p.store[key]

	// If entry doesn't exist or has expired, create a new one
	if !exists || now.After(entry.expiresAt) {
		expiresAt := now.Add(window)
		p.store[key] = &inMemoryEntry{
			count:     1,
			expiresAt: expiresAt,
		}
		return true, 1, expiresAt, nil
	}

	// Entry exists and hasn't expired
	entry.count++

	allowed := entry.count <= maxRequests
	return allowed, entry.count, entry.expiresAt, nil
}

// Close closes the provider (no-op for in-memory)
func (p *InMemoryProvider) Close() error {
	return nil
}

// cleanupExpired periodically removes expired entries
func (p *InMemoryProvider) cleanupExpired() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for key, entry := range p.store {
			if now.After(entry.expiresAt) {
				delete(p.store, key)
			}
		}
		p.mu.Unlock()
	}
}
