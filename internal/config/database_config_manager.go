package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// DatabaseConfigManager implements ConfigManager using a database backend.
type DatabaseConfigManager struct {
	db *gorm.DB
	// Use atomic.Value to store the *models.Config for lock-free reads
	activeConfig atomic.Value
	mu           sync.Mutex
}

func NewDatabaseConfigManager(initialConfig *models.Config) models.ConfigManager {
	cm := &DatabaseConfigManager{
		db: initialConfig.DB,
	}

	// Initialize with provided config
	cm.activeConfig.Store(initialConfig)

	return cm
}

// Init creates the initial config in the database from the current active config,
// but only if the "runtime_config" key does not already exist.
func (cm *DatabaseConfigManager) Init() error {
	var count int64
	if err := cm.db.Model(&models.AuthSettings{}).
		Where("key = ?", "runtime_config").
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check for existing runtime_config: %w", err)
	}
	if count > 0 {
		// Config already exists, just load it
		if err := cm.Load(); err != nil {
			return err
		}
		return nil
	}

	current := cm.GetConfig()
	jsonData, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("failed to marshal initial config: %w", err)
	}

	return cm.db.Create(&models.AuthSettings{
		Key:   "runtime_config",
		Value: jsonData,
	}).Error
}

// GetConfig returns the current active configuration.
func (cm *DatabaseConfigManager) GetConfig() *models.Config {
	return cm.activeConfig.Load().(*models.Config)
}

// Load loads the configuration from the database and updates the active config.
// If the configuration doesn't exist in the database yet, it initializes it from the current config.
func (cm *DatabaseConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var settings models.AuthSettings
	if err := cm.db.First(&settings, "key = ?", "runtime_config").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("runtime_config not found. It needs to be initialized first: %w", err)
		}
		return err
	}

	current := cm.GetConfig()
	// Create a shallow copy to modify
	newCfg := *current

	// Unmarshal JSON from DB into the new config
	// This will overwrite fields present in JSON but keep others (like functions) intact
	if err := json.Unmarshal(settings.Value, &newCfg); err != nil {
		return fmt.Errorf("failed to unmarshal database config: %w", err)
	}

	util.PreserveNonSerializableFieldsOnConfig(&newCfg, current)

	*current = newCfg

	return nil
}

// Update updates a specific configuration value by key (dot notation) and persists it.
func (cm *DatabaseConfigManager) Update(key string, value any) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	current := cm.GetConfig()

	updatedConfig, err := ValidateAndMergeConfig(current, key, value)
	if err != nil {
		return err
	}

	util.PreserveNonSerializableFieldsOnConfig(updatedConfig, current)

	newJSON, err := json.Marshal(updatedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	err = cm.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&models.AuthSettings{
		Key:   "runtime_config",
		Value: newJSON,
	}).Error

	if err != nil {
		return fmt.Errorf("failed to update config in database: %w", err)
	}

	*current = *updatedConfig

	return nil
}

// Watch watches for configuration changes in the database.
// For PostgreSQL, it uses LISTEN/NOTIFY for efficient pub/sub.
// For other databases, it uses polling with a 5-second interval.
func (cm *DatabaseConfigManager) Watch(ctx context.Context) (<-chan *models.Config, error) {
	configChan := make(chan *models.Config)

	// Determine if we're using PostgreSQL
	if isPostgres(cm.db) {
		go cm.watchPostgres(ctx, configChan)
	} else {
		go cm.watchPolling(ctx, configChan)
	}

	return configChan, nil
}

// watchPostgres uses PostgreSQL LISTEN/NOTIFY for efficient configuration watching
func (cm *DatabaseConfigManager) watchPostgres(ctx context.Context, configChan chan<- *models.Config) {
	defer close(configChan)

	// Get raw database connection for LISTEN/NOTIFY
	sqlDB, err := cm.db.DB()
	if err != nil {
		slog.Error("Failed to get database connection for watching", "error", err)
		return
	}

	// Create a new connection for listening
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		slog.Error("Failed to create database connection for watching", "error", err)
		return
	}
	defer conn.Close()

	// Send LISTEN command
	_, err = conn.ExecContext(ctx, "LISTEN config_changed")
	if err != nil {
		slog.Warn("Failed to set up LISTEN, falling back to polling", "error", err)
		cm.watchPolling(ctx, configChan)
		return
	}

	defer func() {
		_, _ = conn.ExecContext(context.Background(), "UNLISTEN config_changed")
	}()

	// For now, use polling as a fallback since Postgres LISTEN with GORM requires
	// more complex connection handling with pq.Listener
	cm.watchPolling(ctx, configChan)
}

// watchPolling polls the database for configuration changes at a regular interval
func (cm *DatabaseConfigManager) watchPolling(ctx context.Context, configChan chan<- *models.Config) {
	defer close(configChan)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastUpdated := time.Now()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			var settings models.AuthSettings
			// Use a session with silent logger to avoid logging expected "record not found" errors
			err := cm.db.Session(&gorm.Session{
				Logger: cm.db.Logger.LogMode(logger.Silent),
			}).First(&settings, "key = ?", "runtime_config").Error

			if err != nil {
				if err != gorm.ErrRecordNotFound {
					slog.Error("Failed to check for config updates", "error", err)
				}
				continue
			}

			if settings.UpdatedAt.After(lastUpdated) {
				lastUpdated = settings.UpdatedAt

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

// isPostgres checks if the database is PostgreSQL
func isPostgres(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	dialector := db.Dialector.Name()
	return strings.ToLower(dialector) == "postgres"
}
