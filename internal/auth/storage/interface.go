package storage

import (
	"context"
	"time"
)

// SecondaryStorage defines an interface for secondary storage operations.
type SecondaryStorage interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value any, ttl *time.Duration) error
	Delete(ctx context.Context, key string) error
}
