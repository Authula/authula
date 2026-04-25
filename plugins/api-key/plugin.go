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
	rootservices "github.com/Authula/authula/services"
)

type ApiKeyPlugin struct {
	config types.ApiKeyPluginConfig
	logger models.Logger
	db     bun.IDB
	ctx    *models.PluginContext
	Api    *API
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
	p.ctx = ctx
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

	var organizationService rootservices.OrganizationService
	if p.config.AllowOrgKeys {
		orgSvc, ok := ctx.ServiceRegistry.Get(models.ServiceOrganization.String()).(rootservices.OrganizationService)
		if !ok {
			return fmt.Errorf("allow_org_keys is enabled but required service %s is not registered", models.ServiceOrganization.String())
		}
		organizationService = orgSvc
	}

	apiKeyRepo := apirepositories.NewBunApiKeyRepository(p.db)
	service := apiservices.NewApiKeyService(p.config, userService, tokenService, organizationService, apiKeyRepo)

	p.Api = NewAPI(service)

	return nil
}

func (p *ApiKeyPlugin) Migrations(provider string) []migrations.Migration {
	return apiKeyMigrationsForProvider(provider)
}

func (p *ApiKeyPlugin) DependsOn() []string {
	return []string{}
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
