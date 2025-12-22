package config

import (
	"log/slog"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// NewConfigManager creates a config manager based on the runtime mode and settings.
// - Library mode: no config manager needed (embedded in API)
// - Standalone mode: uses database-backed configuration
func NewConfigManager(config *models.Config) models.ConfigManager {
	// Library mode doesn't use a config manager
	if config.Mode == models.ModeLibrary {
		slog.Debug("Running in library mode - no config manager needed")
		return nil
	}

	if config.Mode == models.ModeStandalone {
		return NewDatabaseConfigManager(config)
	}

	return nil
}
