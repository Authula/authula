package configmanager

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager/handlers"
	configmanagerservices "github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager/types"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type ConfigManagerPlugin struct {
	config        types.ConfigManagerPluginConfig
	logger        models.Logger
	ctx           *models.PluginContext
	configManager models.ConfigManager
}

func New(config types.ConfigManagerPluginConfig) *ConfigManagerPlugin {
	return &ConfigManagerPlugin{
		config: config,
	}
}

func (p *ConfigManagerPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginConfigManager.String(),
		Version:     "1.0.0",
		Description: "Provides Config Manager API for configuration management",
	}
}

func (p *ConfigManagerPlugin) Config() any {
	return p.config
}

func (p *ConfigManagerPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	if ctx.DB != nil {
		tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(services.TokenService)
		if !ok {
			return fmt.Errorf("token service not found")
		}

		// Create and initialize the config manager
		// Note: We need to get the initial config to pass to ConfigManager
		// Use a temporary closure to capture the current config
		initialConfig := ctx.GetConfig()
		configManager := NewConfigManager(initialConfig, ctx.DB, tokenService)

		configManagerService := configmanagerservices.NewConfigManagerService(p.logger, configManager)

		// Set the callback to notify watchers when config is updated
		if dbConfigManager, ok := configManager.(*DatabaseConfigManager); ok {
			dbConfigManager.SetOnConfigUpdate(func(cfg *models.Config) error {
				return configManagerService.NotifyWatchers(cfg)
			})
		}

		if err := configManager.Init(); err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}
		p.configManager = configManager

		ctx.ServiceRegistry.Register(models.ServiceConfigManager.String(), configManagerService)

		// If PluginRegistry supports SetConfigProvider, set the config provider to use ConfigManager
		if registry, ok := ctx.ServiceRegistry.(interface{ SetConfigProvider(func() *models.Config) }); ok {
			registry.SetConfigProvider(func() *models.Config {
				return p.configManager.GetConfig()
			})
		}
	}

	return nil
}

func (p *ConfigManagerPlugin) Migrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	return GetMigrations(ctx, dbProvider)
}

func (p *ConfigManagerPlugin) Routes() []models.Route {
	if p.ctx == nil || p.configManager == nil {
		return []models.Route{}
	}

	getConfigHandler := &handlers.ConfigManagerGetConfigHandler{
		ConfigManager: p.configManager,
	}

	updateConfigHandler := &handlers.ConfigManagerUpdateConfigHandler{
		ConfigManager: p.configManager,
	}

	return []models.Route{
		{
			Method: "GET",
			Path:   "/config",
			Middleware: []func(http.Handler) http.Handler{
				ConfigManagerAuthMiddleware,
			},
			Handler: http.HandlerFunc(getConfigHandler.Handle),
		},
		{
			Method: "PATCH",
			Path:   "/config",
			Middleware: []func(http.Handler) http.Handler{
				ConfigManagerAuthMiddleware,
			},
			Handler: http.HandlerFunc(updateConfigHandler.Handle),
		},
	}
}

func (p *ConfigManagerPlugin) Close() error {
	return nil
}
