package services

import (
	"context"
	"time"
)

type RateLimiterService interface {
	CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error)
	SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error
	GetRule(ctx context.Context, key string) (window time.Duration, maxRequests int, found bool, err error)
	DeleteRule(ctx context.Context, key string) error
}
