package plugins

import (
	"log/slog"
	"os"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type PluginRegistry struct {
	config    *domain.Config
	pluginCtx *domain.PluginContext
	plugins   []domain.Plugin
}

func NewPluginRegistry(config *domain.Config, eventBus domain.EventBus) *PluginRegistry {
	ctx := &domain.PluginContext{
		Config:   config,
		EventBus: eventBus,
	}

	return &PluginRegistry{
		config:    config,
		pluginCtx: ctx,
		plugins:   make([]domain.Plugin, 0),
	}
}

func (r *PluginRegistry) Register(p domain.Plugin) {
	r.plugins = append(r.plugins, p)
}

func (r *PluginRegistry) InitAll() error {
	for _, plugin := range r.plugins {
		if !plugin.Config().Enabled {
			continue
		}

		if err := plugin.Init(r.pluginCtx); err != nil {
			return err
		}
	}
	return nil
}

func (r *PluginRegistry) RunMigrations() error {
	for _, plugin := range r.plugins {
		if !plugin.Config().Enabled {
			continue
		}

		migrations := plugin.Migrations()
		if len(migrations) > 0 {
			if err := r.config.DB.AutoMigrate(migrations...); err != nil {
				logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
				logger.Error("failed to run plugin migration", "plugin", plugin.Metadata().Name, "error", err)
				return err
			}
		}
	}
	return nil
}

func (r *PluginRegistry) Plugins() []domain.Plugin {
	plugins := make([]domain.Plugin, 0)
	for _, plugin := range r.plugins {
		if !plugin.Config().Enabled {
			continue
		}
		plugins = append(plugins, plugin)
	}
	return plugins
}

func (r *PluginRegistry) CloseAll() {
	for _, plugin := range r.plugins {
		if !plugin.Config().Enabled {
			continue
		}

		if err := plugin.Close(); err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			logger.Error("failed to close plugin", "plugin", plugin.Metadata().Name, "error", err)
		}
	}
}
