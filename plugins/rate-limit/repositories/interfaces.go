package repositories

import (
	"context"
	"time"

	"github.com/Authula/authula/plugins/rate-limit/types"
)

type RateLimitRepository interface {
	GetByKey(ctx context.Context, key string) (*types.RateLimit, error)
	UpdateOrCreate(ctx context.Context, key string, window time.Duration) (*types.RateLimit, error)
	CleanupExpired(ctx context.Context, now time.Time) error
}
