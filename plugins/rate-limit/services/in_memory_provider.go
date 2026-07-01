package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Authula/authula/plugins/rate-limit/types"
)

type InMemoryProvider struct {
	mu              sync.RWMutex
	store           map[string]*inMemoryEntry
	rules           map[string]*inMemoryRule
	cleanupInterval time.Duration
}

type inMemoryEntry struct {
	count     int
	expiresAt time.Time
}

type inMemoryRule struct {
	window      time.Duration
	maxRequests int
}

func NewInMemoryProvider() *InMemoryProvider {
	return NewInMemoryProviderWithConfig(types.MemoryStorageConfig{})
}

func NewInMemoryProviderWithConfig(config types.MemoryStorageConfig) *InMemoryProvider {
	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute
	}

	provider := &InMemoryProvider{
		store:           make(map[string]*inMemoryEntry),
		rules:           make(map[string]*inMemoryRule),
		cleanupInterval: cleanupInterval,
	}

	go provider.cleanupExpired()

	return provider
}

func (p *InMemoryProvider) GetName() string {
	return "memory"
}

func (p *InMemoryProvider) GetValue(ctx context.Context, key string) (any, error) {
	entry, exists := p.store[key]
	if exists {
		rateLimitValue := types.RateLimitValue{
			Count:     entry.count,
			ExpiresAt: entry.expiresAt,
		}
		return rateLimitValue, nil
	}

	return nil, nil
}

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

func (p *InMemoryProvider) SetRule(_ context.Context, key string, window time.Duration, maxRequests int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules[key] = &inMemoryRule{window: window, maxRequests: maxRequests}
	return nil
}

func (p *InMemoryProvider) GetRule(_ context.Context, key string) (time.Duration, int, bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	rule, ok := p.rules[key]
	if !ok {
		return 0, 0, false, nil
	}
	return rule.window, rule.maxRequests, true, nil
}

func (p *InMemoryProvider) DeleteRule(_ context.Context, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.rules, key)
	return nil
}

func (p *InMemoryProvider) Close() error {
	return nil
}

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
