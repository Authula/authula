package ratelimit

import (
	"context"
	"embed"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/rate-limit/types"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type RateLimitPlugin struct {
	config   RateLimitPluginConfig
	ctx      *models.PluginContext
	logger   models.Logger
	handler  *RateLimitHookHandler
	provider types.RateLimitProvider
}

func New(config RateLimitPluginConfig) *RateLimitPlugin {
	config.ApplyDefaults()
	return &RateLimitPlugin{
		config: config,
	}
}

func (p *RateLimitPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginRateLimit.String(),
		Version:     "1.0.0",
		Description: "Provides rate limiting functionality",
	}
}

func (p *RateLimitPlugin) Config() any {
	return p.config
}

func (p *RateLimitPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	p.config.ApplyDefaults()
	p.initProvider(p.ctx)

	return nil
}

// trySecondaryStorage attempts to initialize a provider using the secondary-storage service.
// Returns the provider if successful, or nil if not.
func (p *RateLimitPlugin) trySecondaryStorage() types.RateLimitProvider {
	if p.ctx == nil {
		return nil
	}

	secondaryStorageService, ok := p.ctx.ServiceRegistry.Get(models.ServiceSecondaryStorage.String()).(services.SecondaryStorageService)
	if !ok {
		return nil
	}

	storage := secondaryStorageService.GetStorage()
	if storage == nil {
		return nil
	}

	actualProviderName := secondaryStorageService.GetProviderName()
	provider := NewSecondaryStorageProvider(actualProviderName, storage)

	return provider
}

func (p *RateLimitPlugin) Close() error {
	if p.provider != nil {
		return p.provider.Close()
	}
	return nil
}

func (p *RateLimitPlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(config, p.Metadata().ID, &p.config); err != nil {
		p.logger.Error("failed to parse ratelimit plugin config on update", "error", err)
		return err
	}

	p.config.ApplyDefaults()
	p.initProvider(p.ctx)

	return nil
}

func (p *RateLimitPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

// Migrations implements the PluginWithMigrations interface.
// Only returns migrations if the Provider is configured as "database".
// For "memory" or "redis" providers, no database tables are needed.
func (p *RateLimitPlugin) Migrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	// Check if database provider is configured
	if p.config.Provider != types.RateLimitProviderDatabase {
		return nil, nil // No DB tables needed for memory/redis providers
	}
	return MigrationFS.GetMigrations(ctx, dbProvider)
}

func (p *RateLimitPlugin) initProvider(ctx *models.PluginContext) error {
	if p.config.Provider == types.RateLimitProviderRedis {
		if provider := p.trySecondaryStorage(); provider != nil {
			p.provider = provider
			p.handler = NewRateLimitHookHandler(
				ctx.GetConfig(),
				p.logger,
				p.config,
				p.provider,
			)
			return nil
		}
		p.logger.Warn("Redis provider not available via secondary-storage, falling back to in-memory")
	}

	if p.config.Provider == types.RateLimitProviderDatabase {
		if ctx.DB != nil {
			dbConfig := DatabaseStorageConfig{}
			if p.config.Database != nil {
				dbConfig = *p.config.Database
			}
			dbProvider, err := NewDatabaseProviderWithConfig(ctx.DB, dbConfig)
			if err != nil {
				p.logger.Error("failed to initialize database provider", "error", err)
				p.logger.Warn("falling back to in-memory")
			} else {
				p.provider = dbProvider
				p.handler = NewRateLimitHookHandler(
					ctx.GetConfig(),
					p.logger,
					p.config,
					p.provider,
				)
				return nil
			}
		} else {
			p.logger.Warn("database connection not available, falling back to in-memory")
		}
	}

	memoryConfig := MemoryStorageConfig{}
	if p.config.Memory != nil {
		memoryConfig = *p.config.Memory
	}
	p.provider = NewInMemoryProviderWithConfig(memoryConfig)
	p.handler = NewRateLimitHookHandler(
		ctx.GetConfig(),
		p.logger,
		p.config,
		p.provider,
	)

	return nil
}
