package configmanager

import (
	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

// NewConfigManager creates a config manager based on the runtime mode and settings.
func NewConfigManager(config *models.Config, db bun.IDB, tokenService services.TokenService) models.ConfigManager {
	return NewDatabaseConfigManager(config, db, tokenService)
}
