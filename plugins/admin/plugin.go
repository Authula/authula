package admin

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type AdminPlugin struct {
	config types.AdminPluginConfig
	ctx    *models.PluginContext
	logger models.Logger
	Api    *API
}

func New(config types.AdminPluginConfig) *AdminPlugin {
	config.ApplyDefaults()
	return &AdminPlugin{config: config}
}

func (p *AdminPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginAdmin.String(),
		Version:     "1.0.0",
		Description: "Provides admin operations and RBAC functionality.",
	}
}

func (p *AdminPlugin) Config() any {
	return p.config
}

func (p *AdminPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	repos := repositories.NewAdminRepositories(ctx.DB)
	adminServices := services.NewAdminServices(repos)

	userService, ok := ctx.ServiceRegistry.Get(models.ServiceUser.String()).(rootservices.UserService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceUser.String())
	}

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(rootservices.SessionService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceSession.String())
	}

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceToken.String())
	}

	adminUseCases := usecases.NewAdminUseCases(
		adminServices,
		userService,
		p.config,
		sessionService,
		tokenService,
		ctx.GetConfig().Session.ExpiresIn,
	)
	p.Api = NewAPI(adminUseCases)
	ctx.ServiceRegistry.Register(models.ServiceAdmin.String(), p.Api)

	return nil
}

func (p *AdminPlugin) Migrations(provider string) []migrations.Migration {
	return adminMigrationsForProvider(provider)
}

func (p *AdminPlugin) DependsOn() []string {
	return []string{}
}

func (p *AdminPlugin) Routes() []models.Route {
	if p.Api == nil {
		return []models.Route{}
	}
	return Routes(p.Api)
}

func (p *AdminPlugin) Close() error {
	return nil
}
