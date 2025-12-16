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

// KeyValueStore represents the persistent key-value store table in the database.
// This is a domain model used for secondary storage operations.
type KeyValueStore struct {
	Key       string     `json:"key" gorm:"primaryKey"`
	Value     string     `json:"value"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// SecondaryStorageMemoryOptions holds settings specific to the in-memory storage.
type SecondaryStorageMemoryOptions struct {
	// CleanupInterval controls how often expired entries are cleaned up.
	// If zero, the implementation should use a default.
	CleanupInterval time.Duration
}

// SecondaryStorageDatabaseOptions holds settings specific to the database storage.
type SecondaryStorageDatabaseOptions struct {
	// CleanupInterval controls how often expired entries are cleaned up.
	// If zero, the implementation should use a default.
	CleanupInterval time.Duration
}

// SecondaryStorage defines an interface for secondary storage operations.
type SecondaryStorage interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, ttl *time.Duration) error
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string, ttl *time.Duration) (int, error)
}
