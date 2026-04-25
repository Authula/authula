package types

import (
	"os"
	"time"

	"github.com/Authula/authula/env"
)

type RateLimitProviderType string

const (
	RateLimitProviderInMemory RateLimitProviderType = "memory"
	RateLimitProviderRedis    RateLimitProviderType = "redis"
	RateLimitProviderDatabase RateLimitProviderType = "database"
)

func (r RateLimitProviderType) String() string {
	return string(r)
}

type RateLimitPluginConfig struct {
	Enabled     bool                     `json:"enabled" toml:"enabled"`
	Window      time.Duration            `json:"window" toml:"window"`
	Max         int                      `json:"max" toml:"max"`
	Prefix      string                   `json:"prefix,omitempty" toml:"prefix"`
	CustomRules map[string]RateLimitRule `json:"custom_rules" toml:"custom_rules"`
	Provider    RateLimitProviderType    `json:"provider" toml:"provider"`
	Memory      *MemoryStorageConfig     `json:"memory,omitempty" toml:"memory"`
	Database    *DatabaseStorageConfig   `json:"database,omitempty" toml:"database"`
}

type MemoryStorageConfig struct {
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

type DatabaseStorageConfig struct {
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
}
