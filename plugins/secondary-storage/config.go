package secondarystorage

import (
	"time"
)

type SecondaryStorageProvider string

const (
	SecondaryStorageProviderMemory   SecondaryStorageProvider = "memory"
	SecondaryStorageProviderDatabase SecondaryStorageProvider = "database"
	SecondaryStorageProviderRedis    SecondaryStorageProvider = "redis"
)

func (p SecondaryStorageProvider) String() string {
	return string(p)
}

// SecondaryStoragePluginConfig is the main configuration for the secondary storage plugin
// Defaults to in-memory storage if no provider is configured
type SecondaryStoragePluginConfig struct {
	// Enabled controls whether secondary storage is available
	Enabled bool `json:"enabled" toml:"enabled"`

	// Provider specifies which storage backend to use: "memory", "database", or "redis"
	// Defaults to "memory" if not specified or if the selected provider is not available
	Provider SecondaryStorageProvider `json:"provider" toml:"provider"`

	// Memory provider configuration
	Memory *MemoryStorageConfig `json:"memory" toml:"memory"`

	// Database provider configuration
	Database *DatabaseStorageConfig `json:"database" toml:"database"`

	// Redis provider configuration
	Redis *RedisStorageConfig `json:"redis" toml:"redis"`
}

// MemoryStorageConfig contains configuration for in-memory storage provider
type MemoryStorageConfig struct {
	// CleanupInterval controls how often expired entries are cleaned up
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

// DatabaseStorageConfig contains configuration for database storage provider
type DatabaseStorageConfig struct {
	// CleanupInterval controls how often expired entries are cleaned up
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

// RedisStorageConfig contains configuration for Redis storage provider
type RedisStorageConfig struct {
	// URL is the Redis connection URL (e.g., "redis://[username:password@]host[:port]/[db]")
	URL string `json:"url" toml:"url"`
	// MaxRetries is the maximum number of retries for Redis operations
	MaxRetries int `json:"max_retries" toml:"max_retries"`
	// PoolSize is the connection pool size for Redis
	PoolSize int `json:"pool_size" toml:"pool_size"`
	// PoolTimeout is the timeout for getting a connection from the pool
	PoolTimeout time.Duration `json:"pool_timeout" toml:"pool_timeout"`
}

func (config *SecondaryStoragePluginConfig) ApplyDefaults() {
	if config.Provider == "" {
		config.Provider = SecondaryStorageProviderMemory
	}
}
