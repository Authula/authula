package secondarystorage

import (
	"context"
	"embed"
	"os"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/env"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type SecondaryStoragePlugin struct {
	config  SecondaryStoragePluginConfig
	logger  models.Logger
	storage models.SecondaryStorage
}

// New creates a new SecondaryStoragePlugin with the given configuration
func New(config SecondaryStoragePluginConfig) *SecondaryStoragePlugin {
	config.ApplyDefaults()
	return &SecondaryStoragePlugin{
		config: config,
	}
}

// NewWithStorage creates a SecondaryStoragePlugin with a custom SecondaryStorage implementation
// This allows library mode users to provide their own storage backend
func NewWithStorage(providerName string, storage models.SecondaryStorage) *SecondaryStoragePlugin {
	return &SecondaryStoragePlugin{
		config: SecondaryStoragePluginConfig{
			Provider: SecondaryStorageProvider(providerName),
		},
		storage: storage,
	}
}

func (p *SecondaryStoragePlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginSecondaryStorage.String(),
		Version:     "1.0.0",
		Description: "Provides secondary storage backends (memory, database, Redis) for other plugins",
	}
}

func (p *SecondaryStoragePlugin) Config() any {
	return p.config
}

// Init initializes the secondary storage plugin
// It attempts to initialize the configured provider and falls back to in-memory if that fails
// If a custom storage implementation was provided via NewWithStorage, this becomes a no-op
func (p *SecondaryStoragePlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	p.config.ApplyDefaults()

	// If storage is already set (e.g., via NewWithStorage), skip initialization
	if p.storage != nil {
		return nil
	}

	var err error
	switch p.config.Provider {
	case SecondaryStorageProviderRedis:
		if p.storage, err = p.initRedisProvider(ctx); err != nil {
			p.logger.Warn("failed to initialize Redis provider, falling back to memory", "error", err)
			p.storage = p.initMemoryProvider()
		}

	case SecondaryStorageProviderDatabase:
		p.storage, err = p.initDatabaseProvider(ctx)
		if err != nil {
			p.logger.Warn("failed to initialize database provider, falling back to memory", "error", err)
			p.storage = p.initMemoryProvider()
		}

	case SecondaryStorageProviderMemory:
		fallthrough
	default:
		p.storage = p.initMemoryProvider()
	}

	service := NewSecondaryStorageService(p.config.Provider.String(), p.storage)
	ctx.ServiceRegistry.Register(models.ServiceSecondaryStorage.String(), service)

	return nil
}

// initMemoryProvider initializes the in-memory storage provider
func (p *SecondaryStoragePlugin) initMemoryProvider() models.SecondaryStorage {
	cleanupInterval := 1 * time.Minute
	if p.config.Memory != nil {
		cleanupInterval = p.config.Memory.CleanupInterval
		if cleanupInterval == 0 {
			cleanupInterval = 1 * time.Minute
		}
	}

	return NewMemorySecondaryStorage(MemoryStorageConfig{
		CleanupInterval: cleanupInterval,
	})
}

// initDatabaseProvider initializes the database storage provider
func (p *SecondaryStoragePlugin) initDatabaseProvider(ctx *models.PluginContext) (models.SecondaryStorage, error) {
	databaseConfig := p.config.Database
	if databaseConfig == nil {
		databaseConfig = &DatabaseStorageConfig{
			CleanupInterval: 1 * time.Minute,
		}
	}

	cleanupInterval := databaseConfig.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute
	}

	return NewDatabaseSecondaryStorage(ctx.DB, DatabaseStorageConfig{
		CleanupInterval: cleanupInterval,
	}), nil
}

// initRedisProvider initializes the Redis storage provider
func (p *SecondaryStoragePlugin) initRedisProvider(ctx *models.PluginContext) (models.SecondaryStorage, error) {
	redisConfig := p.config.Redis
	if redisConfig == nil {
		// Provide sensible defaults when config is nil (e.g., removed by Viper due to being empty)
		redisConfig = &RedisStorageConfig{
			MaxRetries:  3,
			PoolSize:    10,
			PoolTimeout: 5 * time.Second,
		}
	}

	url := os.Getenv(env.EnvRedisURL)
	if url == "" {
		url = redisConfig.URL
		if url == "" {
			return nil, ErrRedisConfigURLNotProvided
		}
	}

	return NewRedisSecondaryStorage(RedisSecondaryStorageOptions{
		URL:         url,
		MaxRetries:  redisConfig.MaxRetries,
		PoolSize:    redisConfig.PoolSize,
		PoolTimeout: redisConfig.PoolTimeout,
	})
}

func (p *SecondaryStoragePlugin) Migrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	if p.config.Provider == SecondaryStorageProviderDatabase {
		return GetMigrations(ctx, dbProvider)
	}
	// Return nil to explicitly skip migrations for non-DB providers
	return nil, nil
}

func (p *SecondaryStoragePlugin) Close() error {
	if p.storage != nil {
		return p.storage.Close()
	}
	return nil
}

func (p *SecondaryStoragePlugin) OnConfigUpdate(config *models.Config) error {
	if pluginCfg, ok := config.Plugins[models.PluginSecondaryStorage.String()]; ok {
		if err := util.ParsePluginConfig(pluginCfg, &p.config); err != nil {
			p.logger.Error("failed to parse secondary_storage plugin config on update", "error", err)
			return err
		}
	}

	p.config.ApplyDefaults()

	// Note: Don't reinitialize storage on config update to avoid losing data
	// The provider and connection settings would require a full restart to safely change
	// This is acceptable because secondary storage provider changes are infrequent
	// and typically only happen at deployment/restart time

	return nil
}
