package types

import (
	"context"
	"time"
)

// RateLimitValue contains the data for a rate limited key
type RateLimitValue struct {
	Count     int
	ExpiresAt time.Time
}

// RateLimitCheckRequest contains the information needed to check rate limits
type RateLimitCheckRequest struct {
	ClientIP   string
	Path       string
	HTTPMethod string
}

// RateLimitCheckResponse contains the result of a rate limit check
type RateLimitCheckResponse struct {
	// Allowed indicates whether the request should be allowed
	Allowed bool
	// Limit is the maximum number of requests allowed
	Limit int
	// Window is the time window for the rate limit in seconds
	Window int
	// RetryAfter is the number of seconds to wait before retrying (only set if Allowed is false)
	RetryAfter int
}

// RateLimitProvider defines the interface for rate limit storage backends
// Implementations can use in-memory storage, Redis, database, or any other backend
type RateLimitProvider interface {
	// GetName returns the name of the provider
	GetName() string
	// GetValue returns a key's value
	GetValue(ctx context.Context, key string) (any, error)
	// CheckAndIncrement checks if a request is allowed and increments the counter if so
	// key is the fully-qualified key (with prefix already included)
	// window is the time window for expiration
	// maxRequests is the maximum number of requests allowed in the window
	// Returns: (allowed bool, currentCount int, resetTime time.Time, error)
	CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error)
	// SetRule stores a per-key rate-limit rule without consuming quota.
	SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error
	// GetRule retrieves a stored per-key rule. Returns 0, 0, false, nil when not found.
	GetRule(ctx context.Context, key string) (window time.Duration, maxRequests int, found bool, err error)
	// DeleteRule removes the stored rule for a key.
	DeleteRule(ctx context.Context, key string) error
	// Close closes any resources held by the provider
	Close() error
}
