package plugins

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/migrations"
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

// InitAll initializes all enabled plugins
func (r *PluginRegistry) InitAll() error {
	for _, plugin := range r.plugins {
		pluginID := plugin.Metadata().ID
		cfg := r.configProvider()

		if !util.IsPluginEnabled(cfg, pluginID) {
			r.logger.Debug("plugin disabled, skipping initialization", "plugin", pluginID)
			continue
		}

		ctx := &models.PluginContext{
			DB:              r.db,
			EventBus:        r.eventBus,
			Logger:          r.logger,
			ServiceRegistry: r.serviceRegistry,
			GetConfig:       r.configProvider,
		}

		if err := plugin.Init(ctx); err != nil {
			r.logger.Error("failed to initialize plugin", "plugin", pluginID, "error", err)
			return err
		}

		r.logger.Info("plugin initialized", "plugin", pluginID)
	}

	r.registerConfigWatchers()
	return nil
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

func (r *PluginRegistry) RunMigrations(ctx context.Context) error {
	dbProvider := r.config.Database.Provider

	for _, plugin := range r.plugins {
		pluginID := plugin.Metadata().ID
		cfg := r.configProvider()

		if !util.IsPluginEnabled(cfg, pluginID) {
			r.logger.Debug("plugin disabled, skipping migrations", "plugin", pluginID)
			continue
		}

		migrator, ok := plugin.(models.PluginWithMigrations)
		if !ok {
			continue
		}

		sqlFS, err := migrator.Migrations(ctx, dbProvider)
		if err != nil {
			r.logger.Error("failed to get migrations",
				"plugin", pluginID,
				"error", err,
			)
			return err
		}

		if sqlFS == nil {
			r.logger.Debug("plugin has no migrations", "plugin", pluginID)
			continue
		}

		if err := migrations.RunMigrations(ctx, r.logger, dbProvider, r.db, *sqlFS, "migrations/"+dbProvider); err != nil {
			return err
		}
	}

	return nil
}

// DropMigrations drops database migrations for all enabled plugins
func (r *PluginRegistry) DropMigrations(ctx context.Context) error {
	dbProvider := r.config.Database.Provider

	for i := len(r.plugins) - 1; i >= 0; i-- {
		plugin := r.plugins[i]
		pluginID := plugin.Metadata().ID

		migrator, ok := plugin.(models.PluginWithMigrations)
		if !ok {
			continue
		}

		sqlFS, err := migrator.Migrations(ctx, dbProvider)
		if err != nil {
			r.logger.Error("failed to get migrations",
				"plugin", pluginID,
				"error", err,
			)
			return err
		}

		if sqlFS == nil {
			r.logger.Debug("plugin has no migrations", "plugin", pluginID)
			continue
		}

		if err := migrations.DropMigrations(ctx, r.logger, dbProvider, r.db, *sqlFS, "migrations/"+dbProvider); err != nil {
			return err
		}
	}

	return nil
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
