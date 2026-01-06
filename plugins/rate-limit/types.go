package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/uptrace/bun"
)

type RateLimitProviderType string

const (
	RateLimitProviderInMemory RateLimitProviderType = "memory"
	RateLimitProviderRedis    RateLimitProviderType = "redis"
	RateLimitProviderDatabase RateLimitProviderType = "database"
)

func (r RateLimitProviderType) String() string {
	return string(r)
}

// RateLimitStore holds in-memory rate limit data
type RateLimitStore struct {
	mu    sync.RWMutex
	data  map[string]*RateLimitEntry
	rules map[string]RateLimitRule
}

// RateLimit represents a rate limit entry in the database for Bun ORM
type RateLimit struct {
	bun.BaseModel `bun:"table:rate_limits"`

	Key       string    `json:"key" bun:",pk"`
	Count     int       `json:"count"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RateLimitRule struct {
	// Disable rate limiting for this endpoint entirely
	Disabled bool `json:"disabled" toml:"disabled"`

	// Time window for the rate limit
	Window time.Duration `json:"window" toml:"window"`

	// Max number of requests allowed within the window
	Max int `json:"max" toml:"max"`

	// Optional override for the storage namespace
	Prefix string `json:"prefix,omitempty" toml:"prefix"`
}

// RateLimitEntry tracks requests for a specific key
type RateLimitEntry struct {
	Count     int
	FirstReq  time.Time
	LastReset time.Time
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
	// CheckAndIncrement checks if a request is allowed and increments the counter if so
	// key is the fully-qualified key (with prefix already included)
	// window is the time window for expiration
	// maxRequests is the maximum number of requests allowed in the window
	// Returns: (allowed bool, currentCount int, resetTime time.Time, error)
	CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error)
	// Close closes any resources held by the provider
	Close() error
}
