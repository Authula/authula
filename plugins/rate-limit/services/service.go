package services

import (
	"context"
	"time"

	"github.com/Authula/authula/plugins/rate-limit/types"
	rootservices "github.com/Authula/authula/services"
)

type rateLimiterService struct {
	provider types.RateLimitProvider
}

func NewRateLimiterService(provider types.RateLimitProvider) rootservices.RateLimiterService {
	return &rateLimiterService{provider: provider}
}

func (s *rateLimiterService) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	return s.provider.CheckAndIncrement(ctx, key, window, maxRequests)
}
