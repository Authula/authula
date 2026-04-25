package apikey

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	apirepositories "github.com/Authula/authula/plugins/api-key/repositories"
	apiservices "github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
	orgplugins "github.com/Authula/authula/plugins/organizations"
	rootservices "github.com/Authula/authula/services"
)

type ApiKeyPlugin struct {
	config               types.ApiKeyPluginConfig
	logger               models.Logger
	db                   bun.IDB
	pluginCtx            *models.PluginContext
	accessControlService rootservices.AccessControlService
	rateLimiterService   rootservices.RateLimiterService
	Api                  *API
}

func New(config types.ApiKeyPluginConfig) *ApiKeyPlugin {
	config.ApplyDefaults()
	return &ApiKeyPlugin{config: config}
}

func (p *ApiKeyPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginApiKey.String(),
		Version:     "1.0.0",
		Description: "Provides API key management operations.",
	}
}

func (p *ApiKeyPlugin) Config() any {
	return p.config
}

func (p *ApiKeyPlugin) Init(ctx *models.PluginContext) error {
	p.pluginCtx = ctx
	p.logger = ctx.Logger
	p.db = ctx.DB

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	userService, ok := ctx.ServiceRegistry.Get(models.ServiceUser.String()).(rootservices.UserService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceUser.String())
	}

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceToken.String())
	}

	accessControlService, ok := ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()).(rootservices.AccessControlService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceAccessControl.String())
	}
	p.accessControlService = accessControlService

	rateLimiterService, ok := p.pluginCtx.ServiceRegistry.Get(models.ServiceRateLimit.String()).(rootservices.RateLimiterService)
	if !ok {
		p.logger.Warn("rate limit service is unavailable")
	} else {
		p.rateLimiterService = rateLimiterService
	}

	var organizationService rootservices.OrganizationService
	if p.config.AllowOrgKeys {
		orgSvc, ok := ctx.ServiceRegistry.Get(models.ServiceOrganization.String()).(orgplugins.OrganizationLookupService)
		if !ok {
			return fmt.Errorf("allow_org_keys is enabled but required service %s is not registered", models.ServiceOrganization.String())
		}
		organizationService = orgSvc
	}

	apiKeyRepo := apirepositories.NewBunApiKeyRepository(p.db)
	service := apiservices.NewApiKeyService(p.config, userService, tokenService, accessControlService, rateLimiterService, organizationService, apiKeyRepo)

	p.Api = NewAPI(service)

	return nil
}

func (p *ApiKeyPlugin) Migrations(provider string) []migrations.Migration {
	return apiKeyMigrationsForProvider(provider)
}

func (p *ApiKeyPlugin) DependsOn() []string {
	return []string{models.PluginAccessControl.String()}
}

func (p *ApiKeyPlugin) Routes() []models.Route {
	return Routes(p.Api)
}

func (p *ApiKeyPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *ApiKeyPlugin) Close() error {
	return nil
}
