package session

import (
	"time"
)

type SessionPluginConfig struct {
	Enabled            bool          `json:"enabled" toml:"enabled"`
	AutoCleanup        bool          `json:"auto_cleanup" toml:"auto_cleanup"`
	CleanupInterval    time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user" toml:"max_sessions_per_user"`
}

func (config *SessionPluginConfig) ApplyDefaults() {
	if config.CleanupInterval == 0 {
		config.CleanupInterval = time.Minute
	}
	if config.MaxSessionsPerUser == 0 {
		config.MaxSessionsPerUser = 5
	}
}
