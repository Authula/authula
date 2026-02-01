package ratelimit

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/rate-limit/types"
)

// RateLimitRepository defines the interface for rate limit record persistence
type RateLimitRepository interface {
	// GetByKey retrieves a rate limit record by its key
	GetByKey(ctx context.Context, key string) (*types.RateLimit, error)

	// UpdateOrCreate updates count for an existing record, or creates a new one
	UpdateOrCreate(ctx context.Context, key string, window time.Duration) (*types.RateLimit, error)

	// CleanupExpired removes expired rate limit records
	CleanupExpired(ctx context.Context, now time.Time) error
}
