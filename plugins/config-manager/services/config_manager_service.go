package services

import (
	"context"
	"maps"
	"sync"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// ConfigManagerService implements the ConfigManagerAPI interface.
type ConfigManagerService struct {
	logger        models.Logger
	configManager models.ConfigManager
	watchers      map[string]models.PluginWithConfigWatcher
	watchersMutex sync.RWMutex
}

// NewConfigManagerService creates a new API instance for the config manager plugin.
func NewConfigManagerService(logger models.Logger, configManager models.ConfigManager) services.ConfigManagerService {
	return &ConfigManagerService{
		logger:        logger,
		configManager: configManager,
		watchers:      make(map[string]models.PluginWithConfigWatcher),
	}
}

// GetConfig returns the current global configuration.
func (s *ConfigManagerService) GetConfig(ctx context.Context) (*models.Config, error) {
	if s.configManager == nil {
		return nil, constants.ErrConfigManagerNotAvailable
	}

	config := s.configManager.GetConfig()
	if config == nil {
		return nil, constants.ErrConfigNotAvailable
	}

	return config, nil
}

// GetAuthSettings returns the current authentication settings.
func (s *ConfigManagerService) GetAuthSettings(ctx context.Context) (map[string]any, error) {
	if s.configManager == nil {
		return nil, constants.ErrConfigManagerNotAvailable
	}

	config := s.configManager.GetConfig()
	if config == nil {
		return nil, constants.ErrConfigNotAvailable
	}

	// Return plugin settings from the config
	settings := make(map[string]any)
	if config.Plugins != nil {
		maps.Copy(settings, config.Plugins)
	}

	return settings, nil
}

// GetPluginConfig retrieves the configuration for a specific plugin.
func (s *ConfigManagerService) GetPluginConfig(ctx context.Context, pluginName string) (any, error) {
	if s.configManager == nil {
		return nil, constants.ErrConfigManagerNotAvailable
	}

	config := s.configManager.GetConfig()
	if config == nil {
		return nil, constants.ErrConfigNotAvailable
	}

	if config.Plugins == nil {
		return nil, constants.ErrPluginConfigNotFound
	}

	pluginConfig, exists := config.Plugins[pluginName]
	if !exists {
		return nil, constants.ErrPluginConfigNotFound
	}

	return pluginConfig, nil
}

// RegisterConfigWatcher registers a plugin to receive config update notifications.
func (s *ConfigManagerService) RegisterConfigWatcher(pluginID string, plugin models.PluginWithConfigWatcher) error {
	s.watchersMutex.Lock()
	defer s.watchersMutex.Unlock()

	if _, exists := s.watchers[pluginID]; exists {
		return constants.ErrPluginAlreadyRegistered
	}

	s.watchers[pluginID] = plugin
	return nil
}

// NotifyWatchers calls OnConfigUpdate on all registered config watchers.
// If a watcher returns an error, it is logged but other watchers continue to be notified (fail-open).
func (s *ConfigManagerService) NotifyWatchers(config *models.Config) error {
	s.watchersMutex.RLock()
	// Create a copy of watchers to avoid holding the lock during callbacks
	watchersCopy := make(map[string]models.PluginWithConfigWatcher)
	maps.Copy(watchersCopy, s.watchers)
	s.watchersMutex.RUnlock()

	for pluginID, watcher := range watchersCopy {
		if err := watcher.OnConfigUpdate(config); err != nil {
			s.logger.Error("config watcher error", "plugin", pluginID, "error", err)
			// Continue notifying other watchers (fail-open)
		}
	}

	return nil
}
