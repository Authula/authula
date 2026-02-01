package plugins

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// PluginRegistry manages plugin registration and lifecycle
type PluginRegistry struct {
	config          *models.Config
	logger          models.Logger
	db              bun.IDB
	serviceRegistry models.ServiceRegistry
	eventBus        models.EventBus
	plugins         []models.Plugin
	configProvider  func() *models.Config
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(
	config *models.Config,
	logger models.Logger,
	db bun.IDB,
	serviceRegistry models.ServiceRegistry,
	eventBus models.EventBus,
) *PluginRegistry {
	registry := &PluginRegistry{
		config:          config,
		logger:          logger,
		db:              db,
		serviceRegistry: serviceRegistry,
		eventBus:        eventBus,
		plugins:         make([]models.Plugin, 0),
	}

	registry.configProvider = func() *models.Config {
		return registry.config
	}

	return registry
}

// Register registers a plugin with the registry
func (r *PluginRegistry) Register(p models.Plugin) error {
	pluginID := p.Metadata().ID

	for _, existing := range r.plugins {
		if existing.Metadata().ID == pluginID {
			return fmt.Errorf("plugin with ID %q is already registered", pluginID)
		}
	}

	r.plugins = append(r.plugins, p)
	return nil
}

// SetConfigProvider allows ConfigManager to inject a dynamic config provider
func (r *PluginRegistry) SetConfigProvider(provider func() *models.Config) {
	if provider != nil {
		r.configProvider = provider
	}
}

// InitAll initializes all enabled plugins and runs their migrations
func (r *PluginRegistry) InitAll() error {
	ctx := context.Background()

	for _, plugin := range r.plugins {
		pluginID := plugin.Metadata().ID
		cfg := r.configProvider()

		if !util.IsPluginEnabled(cfg, pluginID, false) {
			r.logger.Debug("plugin disabled, skipping initialization", "plugin", pluginID)
			continue
		}

		pluginCtx := &models.PluginContext{
			DB:              r.db,
			Logger:          r.logger,
			EventBus:        r.eventBus,
			ServiceRegistry: r.serviceRegistry,
			GetConfig:       r.configProvider,
		}

		if err := plugin.Init(pluginCtx); err != nil {
			r.logger.Error("failed to initialize plugin", "plugin", pluginID, "error", err)
			return err
		}

		r.logger.Info("plugin initialized", "plugin", pluginID)
	}

	if err := r.runPluginMigrations(ctx); err != nil {
		r.logger.Error("failed to run some plugin migrations", "error", err)
		// Continue even if some migrations fail
	}

	r.registerConfigWatchers()

	return nil
}

// runPluginMigrations runs migrations for all plugins that implement PluginWithMigrations
func (r *PluginRegistry) runPluginMigrations(ctx context.Context) error {
	return internal.RunPluginMigrations(ctx, r.db, r.logger, r.plugins)
}

// RunMigrations runs migrations for all plugins (for backward compatibility)
func (r *PluginRegistry) RunMigrations(ctx context.Context) error {
	return r.runPluginMigrations(ctx)
}

// DropMigrations drops migrations for all plugins
func (r *PluginRegistry) DropMigrations(ctx context.Context) error {
	return internal.DropPluginMigrations(ctx, r.db, r.logger, r.plugins)
}

// registerConfigWatchers wires config hot-reload
// Plugins that implement PluginWithConfigWatcher are registered to receive
// config update notifications from the ConfigManagerService via the service registry.
func (r *PluginRegistry) registerConfigWatchers() {
	configManagerService, ok := r.serviceRegistry.Get(models.ServiceConfigManager.String()).(services.ConfigManagerService)
	if !ok {
		return
	}

	for _, plugin := range r.plugins {
		pluginID := plugin.Metadata().ID

		watcher, ok := plugin.(models.PluginWithConfigWatcher)
		if !ok {
			continue
		}

		if err := configManagerService.RegisterConfigWatcher(pluginID, watcher); err != nil {
			r.logger.Error("failed to register config watcher", "plugin", pluginID, "error", err)
			continue
		}
	}
}

func (r *PluginRegistry) Plugins() []models.Plugin {
	return r.plugins
}

func (r *PluginRegistry) GetConfig() *models.Config {
	return r.configProvider()
}

func (r *PluginRegistry) CloseAll() {
	for _, plugin := range r.plugins {
		if err := plugin.Close(); err != nil {
			r.logger.Error("failed to close plugin", "plugin", plugin.Metadata().ID, "error", err)
		}
	}
}

func (r *PluginRegistry) GetPlugin(pluginID string) models.Plugin {
	for _, plugin := range r.plugins {
		if plugin.Metadata().ID == pluginID {
			return plugin
		}
	}
	return nil
}
