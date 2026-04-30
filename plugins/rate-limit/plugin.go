package ratelimit

import (
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	pluginservices "github.com/Authula/authula/plugins/rate-limit/services"
	"github.com/Authula/authula/plugins/rate-limit/types"
	rootservices "github.com/Authula/authula/services"
)

type RateLimitPlugin struct {
	config   types.RateLimitPluginConfig
	ctx      *models.PluginContext
	logger   models.Logger
	handler  *RateLimitHookHandler
	provider types.RateLimitProvider
}

func New(config types.RateLimitPluginConfig) *RateLimitPlugin {
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
	if err := p.initProvider(p.ctx); err != nil {
		return err
	}
	if p.provider != nil {
		ctx.ServiceRegistry.Register(models.ServiceRateLimit.String(), pluginservices.NewRateLimiterService(p.provider))
	}

	return nil
}

// trySecondaryStorage attempts to initialize a provider using the secondary-storage service.
// Returns the provider if successful, or nil if not.
func (p *RateLimitPlugin) trySecondaryStorage() types.RateLimitProvider {
	if p.ctx == nil {
		return nil
	}

	secondaryStorageService, ok := p.ctx.ServiceRegistry.Get(models.ServiceSecondaryStorage.String()).(rootservices.SecondaryStorageService)
	if !ok {
		return nil
	}

	storage := secondaryStorageService.GetStorage()
	if storage == nil {
		return nil
	}

	actualProviderName := secondaryStorageService.GetProviderName()
	provider := pluginservices.NewSecondaryStorageProvider(actualProviderName, storage)

	return provider
}
func (p *RateLimitPlugin) Migrations(provider string) []migrations.Migration {
	if p.config.Provider != types.RateLimitProviderDatabase {
		return nil
	}
	return rateLimitMigrationsForProvider(provider)
}

func (p *RateLimitPlugin) DependsOn() []string {
	return nil
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
	if err := p.initProvider(p.ctx); err != nil {
		p.logger.Error("failed to re-initialize provider on config update", "error", err)
		return err
	}

	return nil
}

func (p *RateLimitPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *RateLimitPlugin) initProvider(ctx *models.PluginContext) error {
	if p.config.Provider == types.RateLimitProviderRedis {
		if provider := p.trySecondaryStorage(); provider != nil {
			p.provider = provider
			p.handler = NewRateLimitHookHandler(
				ctx.GetConfig(),
				p.logger,
				p.config,
				pluginservices.NewRateLimiterService(p.provider),
			)
			return nil
		}
		p.logger.Warn("Redis provider not available via secondary-storage, falling back to in-memory")
	}

	if p.config.Provider == types.RateLimitProviderDatabase {
		if ctx.DB != nil {
			dbConfig := types.DatabaseStorageConfig{}
			if p.config.Database != nil {
				dbConfig = *p.config.Database
			}
			dbProvider, err := pluginservices.NewDatabaseProviderWithConfig(ctx.DB, dbConfig)
			if err != nil {
				p.logger.Error("failed to initialize database provider", "error", err)
				p.logger.Warn("falling back to in-memory")
			} else {
				p.provider = dbProvider
				p.handler = NewRateLimitHookHandler(
					ctx.GetConfig(),
					p.logger,
					p.config,
					pluginservices.NewRateLimiterService(p.provider),
				)
				return nil
			}
		} else {
			p.logger.Warn("database connection not available, falling back to in-memory")
		}
	}

	memoryConfig := types.MemoryStorageConfig{}
	if p.config.Memory != nil {
		memoryConfig = *p.config.Memory
	}
	p.provider = pluginservices.NewInMemoryProviderWithConfig(memoryConfig)
	p.handler = NewRateLimitHookHandler(
		ctx.GetConfig(),
		p.logger,
		p.config,
		pluginservices.NewRateLimiterService(p.provider),
	)

	return nil
}
