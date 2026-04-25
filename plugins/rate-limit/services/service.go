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

func (s *rateLimiterService) GetValue(ctx context.Context, key string) (any, error) {
	return s.provider.GetValue(ctx, key)
}

func (s *rateLimiterService) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	return s.provider.CheckAndIncrement(ctx, key, window, maxRequests)
}

func (s *rateLimiterService) SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error {
	return s.provider.SetRule(ctx, key, window, maxRequests)
}

func (s *rateLimiterService) GetRule(ctx context.Context, key string) (*rootservices.RateLimitKeyRule, error) {
	window, maxRequests, found, err := s.provider.GetRule(ctx, key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &rootservices.RateLimitKeyRule{Window: window, MaxRequests: maxRequests}, nil
}

func (s *rateLimiterService) DeleteRule(ctx context.Context, key string) error {
	return s.provider.DeleteRule(ctx, key)
}
