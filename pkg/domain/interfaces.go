package domain

import (
	"context"
	"time"
)

// SecondaryStorage defines an interface for secondary storage operations.
type SecondaryStorage interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, ttl *time.Duration) error
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string, ttl *time.Duration) (int, error)
}
