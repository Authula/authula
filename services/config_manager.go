package services

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type ConfigManagerService interface {
	GetConfig(ctx context.Context) (*models.Config, error)
	GetAuthSettings(ctx context.Context) (map[string]any, error)
	GetPluginConfig(ctx context.Context, pluginName string) (any, error)
	RegisterConfigWatcher(pluginID string, plugin models.PluginWithConfigWatcher) error
	NotifyWatchers(config *models.Config) error
}
