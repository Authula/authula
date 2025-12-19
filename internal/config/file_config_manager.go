package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// FileConfigManager implements ConfigManager using a file-based backend (TOML).
type FileConfigManager struct {
	configPath   string
	activeConfig atomic.Value
	mu           sync.Mutex
}

// NewFileConfigManager creates a new FileConfigManager that loads TOML configuration from a file.
func NewFileConfigManager(initialConfig *models.Config) models.ConfigManager {
	configPath := os.Getenv("GO_BETTER_AUTH_CONFIG_PATH")
	if configPath == "" {
		panic("GO_BETTER_AUTH_CONFIG_PATH environment variable is not set")
	}

	// Expand environment variables in the path
	configPath = os.ExpandEnv(configPath)

	if _, err := toml.DecodeFile(configPath, initialConfig); err != nil {
		panic(fmt.Sprintf("Failed to parse TOML config from %s: %v", configPath, err))
	}

	cm := &FileConfigManager{
		configPath: configPath,
	}

	if err := cm.Load(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Preserve non-serializable fields from the initial config (like DB connection)
	loadedConfig := cm.GetConfig()
	if loadedConfig != nil {
		util.PreserveNonSerializableFieldsOnConfig(loadedConfig, initialConfig)
		cm.activeConfig.Store(loadedConfig)
	}

	return cm
}

// GetConfig returns the current active configuration.
func (cm *FileConfigManager) GetConfig() *models.Config {
	config := cm.activeConfig.Load()
	if config == nil {
		return nil
	}
	return config.(*models.Config)
}

// Load loads the configuration from the TOML file.
func (cm *FileConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var config models.Config

	if _, err := os.Stat(cm.configPath); err == nil {
		if _, err := toml.DecodeFile(cm.configPath, &config); err != nil {
			return fmt.Errorf("failed to parse TOML config: %w", err)
		}
	}

	cm.activeConfig.Store(&config)
	return nil
}

// Update updates a configuration field and persists it to the TOML file.
// It validates the updated configuration using the validator before persisting.
func (cm *FileConfigManager) Update(key string, value any) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Get the current config
	current := cm.activeConfig.Load()
	if current == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Validate and merge the config
	updatedConfig, err := ValidateAndMergeConfig(current.(*models.Config), key, value)
	if err != nil {
		return err
	}

	util.PreserveNonSerializableFieldsOnConfig(updatedConfig, current.(*models.Config))

	// Marshal the updated config to TOML
	data, err := tomlMarshal(updatedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config to TOML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Update the active config
	cm.activeConfig.Store(updatedConfig)

	slog.Info("Configuration updated", "key", key, "value", value)
	return nil
}

// Watch watches for changes to the configuration file and sends updates on the returned channel.
// It uses fsnotify to detect file system events and reloads the configuration when changes are detected.
func (cm *FileConfigManager) Watch(ctx context.Context) (<-chan *models.Config, error) {
	// Create a file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Get the directory containing the config file
	configDir := filepath.Dir(cm.configPath)
	if configDir == "" || configDir == "." {
		// If no directory specified, use current directory
		configDir = "."
	}

	// Watch the configuration file's directory
	err = watcher.Add(configDir)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch config directory: %w", err)
	}

	configChan := make(chan *models.Config)

	go func() {
		defer watcher.Close()
		defer close(configChan)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Check if this is the config file we care about
				if !isConfigFile(event.Name, cm.configPath) {
					continue
				}

				// Only react to write and create events
				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
					continue
				}

				// Add a small delay to ensure the file is fully written
				time.Sleep(100 * time.Millisecond)

				// Reload the configuration
				if err := cm.Load(); err != nil {
					slog.Error("Failed to reload config file", "error", err)
					continue
				}

				// Send the updated config on the channel
				config := cm.GetConfig()
				select {
				case configChan <- config:
				case <-ctx.Done():
					return
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("File watcher error", "error", err)
			}
		}
	}()

	return configChan, nil
}

// isConfigFile checks if a file path matches the config file path
func isConfigFile(filePath, configPath string) bool {
	// Handle both absolute and relative paths
	absFile := filePath
	absConfig := configPath

	// Try to get absolute paths for comparison
	if abspath, err := filepath.Abs(filePath); err == nil {
		absFile = abspath
	}
	if abspath, err := filepath.Abs(configPath); err == nil {
		absConfig = abspath
	}

	return absFile == absConfig || filePath == configPath
}

// tomlMarshal marshals a Config struct to TOML bytes using the toml package.
func tomlMarshal(config *models.Config) ([]byte, error) {
	var buf strings.Builder
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(config); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}
