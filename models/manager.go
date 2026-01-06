package models

import "context"

type ConfigManager interface {
	// Init initializes the config manager, creating initial config if necessary
	Init() error
	// GetConfig returns the current configuration
	GetConfig() *Config
	// Load reloads the configuration from the storage
	Load() error
	// Update updates a specific configuration key with a new value
	Update(key string, value any) error
	// UpdateWithResult updates config and returns the updated config to avoid redundant GetConfig() calls
	UpdateWithResult(key string, value any, result **Config) error
	// Watch returns a channel that emits config updates
	Watch(ctx context.Context) (<-chan *Config, error)
}
