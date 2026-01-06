package configmanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/config-manager/repositories"
	"github.com/GoBetterAuth/go-better-auth/plugins/config-manager/types"
	"github.com/GoBetterAuth/go-better-auth/services"
)

// DatabaseConfigManager implements ConfigManager using a database backend.
type DatabaseConfigManager struct {
	db           bun.IDB
	tokenService services.TokenService
	repository   repositories.AuthSettingsRepository
	encryptor    *ConfigEncryptor
	// Use atomic.Value to store the *models.Config for lock-free reads
	activeConfig atomic.Value
	mu           sync.Mutex
	// onConfigUpdate is called when config is updated to notify watchers
	onConfigUpdate func(config *models.Config) error
	// encryptor handles encryption/decryption of sensitive fields
}

func NewDatabaseConfigManager(initialConfig *models.Config, db bun.IDB, tokenService services.TokenService) models.ConfigManager {
	cm := &DatabaseConfigManager{
		db:           db,
		repository:   repositories.NewBunConfigManagerRepository(db),
		encryptor:    NewConfigEncryptor(initialConfig.Secret, tokenService),
		tokenService: tokenService,
	}

	// Initialize with provided config
	cm.activeConfig.Store(initialConfig)

	return cm
}

// SetOnConfigUpdate sets a callback function to be called when config is updated.
// This is used to notify config watchers (plugins that implement PluginWithConfigWatcher).
func (cm *DatabaseConfigManager) SetOnConfigUpdate(callback func(config *models.Config) error) {
	cm.onConfigUpdate = callback
}

// Init creates the initial config in the database from the current active config,
// but only if the "runtime_config" key does not already exist.
func (cm *DatabaseConfigManager) Init() error {
	existing, err := cm.repository.GetByKey(context.Background(), "runtime_config")
	if err != nil {
		return fmt.Errorf("failed to check for existing runtime_config: %w", err)
	}

	if existing != nil {
		// Config already exists, just load it
		if err := cm.Load(); err != nil {
			return err
		}
		return nil
	}

	current := cm.GetConfig()
	jsonData, err := cm.encryptor.EncryptConfig(current)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	return cm.repository.Upsert(context.Background(), &types.AuthSettings{
		Key:   "runtime_config",
		Value: jsonData,
	})
}

// GetConfig returns the current active configuration.
func (cm *DatabaseConfigManager) GetConfig() *models.Config {
	return cm.activeConfig.Load().(*models.Config)
}

// Load loads the configuration from the database and updates the active config.
// If the configuration doesn't exist in the database yet, it initializes it from the current config.
// After loading, it notifies any registered config watchers.
func (cm *DatabaseConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	settings, err := cm.repository.GetByKey(context.Background(), "runtime_config")
	if err != nil {
		return err
	}

	if settings == nil {
		return fmt.Errorf("runtime_config not found. It needs to be initialized first")
	}

	// Decrypt config from database
	newCfg, err := cm.encryptor.DecryptConfig(settings.Value)
	if err != nil {
		// Decryption failed, might be plaintext config or corrupted encrypted data
		// Try to unmarshal as plaintext and then encrypt it
		slog.Warn("Failed to decrypt config, attempting to load as plaintext or reinitializing", "error", err)
		var plaintextCfg models.Config
		if err := json.Unmarshal(settings.Value, &plaintextCfg); err != nil {
			slog.Warn("Could not load config as plaintext either, reinitializing from current config", "error", err)
			// Both decryption and plaintext parsing failed
			// This likely means the config was corrupted during encryption (e.g., with old buggy nonce format)
			// Reinitialize from the current active config and save it
			newCfg = cm.GetConfig()
			encryptedJSON, encErr := cm.encryptor.EncryptConfig(newCfg)
			if encErr != nil {
				return fmt.Errorf("failed to encrypt config during reinitialization: %w", encErr)
			}
			if saveErr := cm.repository.Upsert(context.Background(), &types.AuthSettings{
				Key:   "runtime_config",
				Value: encryptedJSON,
			}); saveErr != nil {
				return fmt.Errorf("failed to save reinitialized config: %w", saveErr)
			}
			slog.Info("Config reinitialized and saved")
			cm.activeConfig.Store(newCfg)
			return nil
		}
		// Successfully loaded plaintext, use it as the config
		newCfg = &plaintextCfg

		// Now encrypt and save it for future use
		encryptedJSON, encErr := cm.encryptor.EncryptConfig(newCfg)
		if encErr != nil {
			slog.Error("Failed to encrypt plaintext config during migration", "error", encErr)
			// Don't fail here, just continue with plaintext
		} else {
			// Save the encrypted version
			if saveErr := cm.repository.Upsert(context.Background(), &types.AuthSettings{
				Key:   "runtime_config",
				Value: encryptedJSON,
			}); saveErr != nil {
				slog.Error("Failed to save encrypted config during migration", "error", saveErr)
				// Don't fail here, just continue
			}
		}
	}

	cm.activeConfig.Store(newCfg)

	// Notify watchers after config is updated
	if cm.onConfigUpdate != nil {
		if err := cm.onConfigUpdate(newCfg); err != nil {
			slog.Error("Failed to notify config watchers", "error", err)
			// Don't return error - watchers should not block config loading
		}
	}

	return nil
}

// Update updates a specific configuration value by key (dot notation) and persists it.
// After updating the database, it notifies any registered config watchers.
func (cm *DatabaseConfigManager) Update(key string, value any) error {
	_, err := cm.updateInternal(key, value)
	return err
}

// UpdateWithResult updates config and sets the result pointer to the updated config.
// This allows callers to avoid redundant GetConfig() calls.
func (cm *DatabaseConfigManager) UpdateWithResult(key string, value any, result **models.Config) error {
	updatedConfig, err := cm.updateInternal(key, value)
	if err == nil && result != nil {
		*result = updatedConfig
	}
	return err
}

// updateInternal contains the common update logic and returns the updated config.
func (cm *DatabaseConfigManager) updateInternal(key string, value any) (*models.Config, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	current := cm.GetConfig()

	updatedConfig, err := ValidateAndMergeConfig(current, key, value)
	if err != nil {
		return nil, err
	}

	// Optimize: Use selective field encryption instead of full config encryption
	// This avoids re-encrypting all sensitive fields when only one is being updated
	newJSON, err := cm.encryptor.EncryptConfigSelectively(current, updatedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt config: %w", err)
	}

	err = cm.repository.Upsert(context.Background(), &types.AuthSettings{
		Key:   "runtime_config",
		Value: newJSON,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update config in database: %w", err)
	}

	cm.activeConfig.Store(updatedConfig)

	// Notify watchers after config is updated
	if cm.onConfigUpdate != nil {
		if err := cm.onConfigUpdate(updatedConfig); err != nil {
			slog.Error("Failed to notify config watchers", "error", err)
			// Don't return error - watchers should not block config updates
		}
	}

	return updatedConfig, nil
}

// Watch watches for configuration changes in the database.
// It uses polling with a 5-second interval to check for configuration version updates.
func (cm *DatabaseConfigManager) Watch(ctx context.Context) (<-chan *models.Config, error) {
	configChan := make(chan *models.Config)
	go cm.watchPolling(ctx, configChan)
	return configChan, nil
}

// watchPolling polls the database for configuration changes at a regular interval
func (cm *DatabaseConfigManager) watchPolling(ctx context.Context, configChan chan<- *models.Config) {
	defer close(configChan)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastVersion int64
	// Initialize lastVersion from the current database state
	var initialSettings types.AuthSettings
	err := cm.db.NewSelect().
		Model(&initialSettings).
		Where("key = ?", "runtime_config").
		Scan(ctx)
	if err == nil {
		lastVersion = initialSettings.ConfigVersion
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			var settings types.AuthSettings
			err := cm.db.NewSelect().
				Model(&settings).
				Where("key = ?", "runtime_config").
				Scan(ctx)

			if err != nil {
				if err == sql.ErrNoRows {
					slog.Error("Failed to check for config updates", "error", err)
				}
				continue
			}

			if settings.ConfigVersion > lastVersion {
				lastVersion = settings.ConfigVersion

				if err := cm.Load(); err != nil {
					slog.Error("Failed to reload config from database", "error", err)
					continue
				}

				config := cm.GetConfig()
				select {
				case configChan <- config:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
