package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/cleanup"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	apiconstants "github.com/Authula/authula/plugins/api-key/constants"
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
	userService          rootservices.UserService
	organizationService  rootservices.OrganizationService
	Api                  *API
	stopCleanup          chan struct{}
	done                 chan struct{}
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
	p.userService = userService

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceToken.String())
	}

	accessControlService, ok := ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()).(rootservices.AccessControlService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceAccessControl.String())
	}
	p.accessControlService = accessControlService
	if err := p.ensurePermissions(); err != nil {
		return err
	}

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
	p.organizationService = organizationService

	apiKeyRepo := apirepositories.NewBunApiKeyRepository(p.db)
	authorizer := rootservices.NewDefaultAuthorizer()
	service := apiservices.NewApiKeyService(p.config, userService, tokenService, accessControlService, rateLimiterService, organizationService, authorizer, apiKeyRepo)

	p.Api = NewAPI(service)

	if p.config.AutoCleanup {
		p.stopCleanup = make(chan struct{})
		p.done = make(chan struct{})
		go p.runCleanupLoop(p.config.CleanupInterval)
	}

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
	if p.config.AutoCleanup {
		close(p.stopCleanup)
		<-p.done
	}
	return nil
}

func (p *ApiKeyPlugin) runCleanupLoop(interval time.Duration) {
	cleanup.RunLoop(p.stopCleanup, p.done, interval, func(ctx context.Context) {
		if err := p.Api.DeleteExpired(ctx); err != nil {
			p.logger.Error("api key expired cleanup failed", "error", err)
		}
	})
}

func (p *ApiKeyPlugin) ensurePermissions() error {
	if err := p.accessControlService.EnsurePermissions(context.Background(), []rootservices.PermissionDefinition{
		{Key: apiconstants.All, Description: "All organization API key permissions"},
		{Key: apiconstants.OrgApiKeyCreate, Description: "Create API keys for the organization"},
		{Key: apiconstants.OrgApiKeyList, Description: "List API keys for the organization"},
		{Key: apiconstants.OrgApiKeyRead, Description: "Read an API key for the organization"},
		{Key: apiconstants.OrgApiKeyUpdate, Description: "Update an API key for the organization"},
		{Key: apiconstants.OrgApiKeyDelete, Description: "Delete an API key for the organization"},
	}); err != nil {
		return fmt.Errorf("failed to ensure api key permissions: %w", err)
	}
	return nil
}
