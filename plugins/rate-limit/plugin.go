package ratelimit

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	pluginservices "github.com/Authula/authula/plugins/rate-limit/services"
	"github.com/Authula/authula/plugins/rate-limit/types"
	"github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type RateLimitPlugin struct {
	pluginConfig types.RateLimitPluginConfig
	logger       models.Logger
	pluginCtx    *models.PluginContext
	provider     types.RateLimitProvider
}

func New(config types.RateLimitPluginConfig) *RateLimitPlugin {
	config.ApplyDefaults()
	return &RateLimitPlugin{
		pluginConfig: config,
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
	return p.pluginConfig
}

func (p *RateLimitPlugin) Init(pluginCtx *models.PluginContext) error {
	p.pluginCtx = pluginCtx
	p.logger = pluginCtx.Logger

	if err := util.LoadPluginConfig(pluginCtx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
	}

	p.pluginConfig.ApplyDefaults()
	if err := p.initProvider(p.pluginCtx); err != nil {
		return err
	}

	rateLimiterService := pluginservices.NewRateLimiterService(p.provider)
	pluginCtx.ServiceRegistry.Register(models.ServiceRateLimit.String(), rateLimiterService)

	return nil
}

// trySecondaryStorage attempts to initialize a provider using the secondary-storage service.
// Returns the provider if successful, or nil if not.
func (p *RateLimitPlugin) trySecondaryStorage() types.RateLimitProvider {
	secondaryStorageService, ok := p.pluginCtx.ServiceRegistry.Get(models.ServiceSecondaryStorage.String()).(services.SecondaryStorageService)
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
	if p.pluginConfig.Provider != types.RateLimitProviderDatabase {
		return nil
	}
	return ratelimitMigrationsForProvider(provider)
}

func (p *RateLimitPlugin) DependsOn() []string {
	return nil
}

func (p *RateLimitPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *RateLimitPlugin) initProvider(pluginCtx *models.PluginContext) error {
	switch p.pluginConfig.Provider {
	case types.RateLimitProviderRedis:
		if provider := p.trySecondaryStorage(); provider != nil {
			p.provider = provider
			return nil
		}
		p.logger.Warn("Redis provider not available via secondary-storage, falling back to in-memory")
	case types.RateLimitProviderDatabase:
		if pluginCtx.DB != nil {
			dbConfig := types.DatabaseStorageConfig{}
			if p.pluginConfig.Database != nil {
				dbConfig = *p.pluginConfig.Database
			}
			dbProvider, err := pluginservices.NewDatabaseProviderWithConfig(pluginCtx.DB, dbConfig)
			if err != nil {
				p.logger.Error("failed to initialize database provider", "error", err)
				p.logger.Warn("falling back to in-memory")
			} else {
				p.provider = dbProvider
				return nil
			}
		} else {
			p.logger.Warn("database connection not available, falling back to in-memory")
		}
	}

	memoryConfig := types.MemoryStorageConfig{}
	if p.pluginConfig.Memory != nil {
		memoryConfig = *p.pluginConfig.Memory
	}
	p.provider = pluginservices.NewInMemoryProviderWithConfig(memoryConfig)

	return nil
}

func (p *RateLimitPlugin) Close() error {
	return p.provider.Close()
}
