package plugins

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/migrationmanager"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/util"
)

// PluginRegistry manages plugin registration and lifecycle
type PluginRegistry struct {
	config           *models.Config
	logger           models.Logger
	db               bun.IDB
	migrationManager *migrationmanager.Manager
	serviceRegistry  models.ServiceRegistry
	eventBus         models.EventBus
	plugins          []models.Plugin
	configProvider   func() *models.Config
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(
	config *models.Config,
	logger models.Logger,
	db bun.IDB,
	migrationManager *migrationmanager.Manager,
	serviceRegistry models.ServiceRegistry,
	eventBus models.EventBus,
) *PluginRegistry {
	registry := &PluginRegistry{
		config:           config,
		logger:           logger,
		db:               db,
		migrationManager: migrationManager,
		serviceRegistry:  serviceRegistry,
		eventBus:         eventBus,
		plugins:          make([]models.Plugin, 0),
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

// InitAll initializes all enabled plugins in dependency order.
func (r *PluginRegistry) InitAll() error {
	cfg := r.configProvider()
	ordered, err := r.pluginsInDependencyOrder(cfg)
	if err != nil {
		return err
	}

	for _, plugin := range ordered {
		pluginID := plugin.Metadata().ID

		ctx := &models.PluginContext{
			DB:              r.db,
			EventBus:        r.eventBus,
			Logger:          r.logger,
			ServiceRegistry: r.serviceRegistry,
			PluginRegistry:  r,
			GetConfig:       r.configProvider,
		}

		if err := plugin.Init(ctx); err != nil {
			r.logger.Error("failed to initialize plugin", "plugin", pluginID, "error", err)
			return err
		}

		r.logger.Info("plugin initialized", "plugin", pluginID)
	}

	return nil
}

// pluginsInDependencyOrder returns the enabled plugins sorted so that every
// plugin appears after its dependencies. Registration order breaks ties, which
// keeps initialization deterministic.
func (r *PluginRegistry) pluginsInDependencyOrder(cfg *models.Config) ([]models.Plugin, error) {
	position := make(map[string]int, len(r.plugins))
	enabled := make(map[string]models.Plugin, len(r.plugins))

	for i, plugin := range r.plugins {
		pluginID := plugin.Metadata().ID
		position[pluginID] = i
		if !util.IsPluginEnabled(cfg, pluginID) {
			r.logger.Debug("plugin disabled, skipping initialization", "plugin", pluginID)
			continue
		}
		enabled[pluginID] = plugin
	}

	indegree := make(map[string]int, len(enabled))
	graph := make(map[string][]string, len(enabled))
	for pluginID := range enabled {
		indegree[pluginID] = 0
	}

	for _, plugin := range r.plugins {
		pluginID := plugin.Metadata().ID
		if _, ok := enabled[pluginID]; !ok {
			continue
		}

		withDependencies, ok := plugin.(models.PluginWithDependencies)
		if !ok {
			continue
		}

		seen := make(map[string]struct{})
		for _, dependency := range withDependencies.DependsOn() {
			dependency = strings.TrimSpace(dependency)
			if dependency == "" || dependency == migrations.CorePluginID {
				continue
			}
			if _, ok := seen[dependency]; ok {
				continue
			}
			seen[dependency] = struct{}{}

			if _, ok := enabled[dependency]; !ok {
				return nil, fmt.Errorf("plugin %q requires plugin %q to be enabled", pluginID, dependency)
			}

			indegree[pluginID]++
			graph[dependency] = append(graph[dependency], pluginID)
		}
	}

	ordered := make([]models.Plugin, 0, len(enabled))
	done := make(map[string]bool, len(enabled))

	for len(ordered) < len(enabled) {
		next := ""
		for pluginID := range enabled {
			if done[pluginID] || indegree[pluginID] != 0 {
				continue
			}
			if next == "" || position[pluginID] < position[next] {
				next = pluginID
			}
		}

		if next == "" {
			remaining := make([]string, 0, len(enabled)-len(ordered))
			for pluginID := range enabled {
				if !done[pluginID] {
					remaining = append(remaining, pluginID)
				}
			}
			sort.Strings(remaining)
			for i, pluginID := range remaining {
				remaining[i] = fmt.Sprintf("%q", pluginID)
			}
			return nil, fmt.Errorf("plugin dependency cycle detected among %s", strings.Join(remaining, ", "))
		}

		done[next] = true
		ordered = append(ordered, enabled[next])

		for _, dependent := range graph[next] {
			indegree[dependent]--
		}
	}

	return ordered, nil
}

func (r *PluginRegistry) RunMigrations(ctx context.Context) error {
	if r.migrationManager == nil || r.migrationManager.Migrator() == nil {
		return nil
	}

	cfg := r.configProvider()
	if cfg == nil {
		return fmt.Errorf("config provider returned nil config")
	}

	dbProvider := cfg.Database.Provider
	selector := func(plugin models.Plugin) bool {
		pluginID := plugin.Metadata().ID
		if !util.IsPluginEnabled(cfg, pluginID) {
			r.logger.Debug("plugin disabled, skipping migrations", "plugin", pluginID)
			return false
		}
		return true
	}

	return r.migrationManager.RunPlugins(ctx, dbProvider, r.plugins, selector)
}

// DropMigrations rolls back migrations for all registered plugins regardless of enablement state.
func (r *PluginRegistry) DropMigrations(ctx context.Context) error {
	if r.migrationManager == nil || r.migrationManager.Migrator() == nil {
		return nil
	}

	cfg := r.configProvider()
	if cfg == nil {
		return fmt.Errorf("config provider returned nil config")
	}

	return r.migrationManager.DropPlugins(ctx, cfg.Database.Provider, r.plugins, nil)
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
