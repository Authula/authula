package ratelimit

import (
	"os"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/env"
)

type RateLimitPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`

	// Time window for the rate limit
	Window time.Duration `json:"window" toml:"window"`

	// Max number of requests allowed within the window
	Max int `json:"max" toml:"max"`

	// Optional override for the storage namespace prefix
	// Other plugins may also use their own prefixes when accessing secondary storage
	Prefix string `json:"prefix,omitempty" toml:"prefix"`

	// Custom rules for specific paths or methods
	CustomRules map[string]RateLimitRule `json:"custom_rules" toml:"custom_rules"`

	// Provider specifies which rate limit backend to use
	// Options: "memory", "redis", "database"
	// If not specified or if the selected provider is unavailable, defaults to "memory"
	//
	// Note: rate limit plugin can use either its own provider (if specified) or the
	// secondary-storage plugin if available. If secondary-storage plugin is configured
	// and enabled, that takes precedence over the provider selection here.
	Provider RateLimitProviderType `json:"provider" toml:"provider"`

	// Memory contains configuration options for in-memory storage
	Memory *MemoryStorageConfig `json:"memory,omitempty" toml:"memory"`

	// Database contains configuration options for database storage
	Database *DatabaseStorageConfig `json:"database,omitempty" toml:"database"`
}

// MemoryStorageConfig contains configuration options for in-memory rate limit storage
type MemoryStorageConfig struct {
	// CleanupInterval specifies how often to remove expired entries
	// Defaults to 1 minute if not specified
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

// DatabaseStorageConfig contains configuration options for database rate limit storage
type DatabaseStorageConfig struct {
	// CleanupInterval specifies how often to remove expired entries from the database
	// Defaults to 1 minute if not specified
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

func (config *RateLimitPluginConfig) ApplyDefaults() {
	environment := os.Getenv(env.EnvGoEnvironment)
	if environment == "production" {
		config.Enabled = true
	}
	if config.Window == 0 {
		config.Window = 1 * time.Minute
	}
	if config.Max == 0 {
		config.Max = 100
	}
	if config.Prefix == "" {
		config.Prefix = "ratelimit:"
	}
	if config.CustomRules == nil {
		config.CustomRules = make(map[string]RateLimitRule)
	}
	// The provider can only be changed via a server restart.
	// TODO: in the future, we need to look into enabling this without a server restart.
}
