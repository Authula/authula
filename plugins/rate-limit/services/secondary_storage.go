package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Authula/authula/models"
)

const ruleKeyPrefix = "rule:"

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

type storedRule struct {
	WindowSeconds int
	MaxRequests   int
}

// SetRule stores a per-key rate-limit rule without consuming quota.
// The rule is persisted under the key prefix "rule:" with no TTL.
func (p *SecondaryStorageProvider) SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error {
	rule := storedRule{
		WindowSeconds: int(window.Seconds()),
		MaxRequests:   maxRequests,
	}
	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}
	return p.storage.Set(ctx, ruleKeyPrefix+key, data, nil)
}

// GetRule retrieves a stored per-key rule. Returns 0, 0, false, nil when not found.
func (p *SecondaryStorageProvider) GetRule(ctx context.Context, key string) (time.Duration, int, bool, error) {
	raw, err := p.storage.Get(ctx, ruleKeyPrefix+key)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to get rule: %w", err)
	}
	if raw == nil {
		return 0, 0, false, nil
	}

	var data []byte
	switch v := raw.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return 0, 0, false, fmt.Errorf("unexpected rule value type %T", raw)
	}

	var rule storedRule
	if err := json.Unmarshal(data, &rule); err != nil {
		return 0, 0, false, fmt.Errorf("failed to unmarshal rule: %w", err)
	}
	return time.Duration(rule.WindowSeconds) * time.Second, rule.MaxRequests, true, nil
}

// DeleteRule removes the stored rule for a key.
func (p *SecondaryStorageProvider) DeleteRule(ctx context.Context, key string) error {
	return p.storage.Delete(ctx, ruleKeyPrefix+key)
}

// Close closes the provider (no-op since we don't own the storage)
func (p *SecondaryStorageProvider) Close() error {
	return nil
}
