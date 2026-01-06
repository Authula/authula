package models

import (
	"context"
	"time"
)

type SecondaryStorageType string

const (
	SecondaryStorageTypeMemory   SecondaryStorageType = "memory"
	SecondaryStorageTypeDatabase SecondaryStorageType = "database"
	SecondaryStorageTypeCustom   SecondaryStorageType = "custom"
)

// SecondaryStorage defines an interface for secondary storage operations.
type SecondaryStorage interface {
	// Get retrieves the value associated with the given key.
	Get(ctx context.Context, key string) (any, error)
	// Set stores a value with an optional time-to-live (TTL).
	Set(ctx context.Context, key string, value any, ttl *time.Duration) error
	// Delete removes the value associated with the given key.
	Delete(ctx context.Context, key string) error
	// Incr increments an integer value associated with the given key.
	Incr(ctx context.Context, key string, ttl *time.Duration) (int, error)
	// TTL retrieves the time-to-live (TTL) for the given key.
	TTL(ctx context.Context, key string) (*time.Duration, error)
	// Close closes the storage and releases any resources.
	Close() error
}
