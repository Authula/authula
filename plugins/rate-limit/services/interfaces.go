package services

import (
	"context"
	"time"
)

type RateLimiterService interface {
	CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error)
}
