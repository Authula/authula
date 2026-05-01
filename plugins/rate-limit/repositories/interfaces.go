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
	SetRule(ctx context.Context, key string, windowSeconds int, maxRequests int) error
	GetRule(ctx context.Context, key string) (*types.RateLimitRuleRecord, error)
	DeleteRule(ctx context.Context, key string) error
}
