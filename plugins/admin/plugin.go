package admin

import (
	"fmt"

	coreinternalrepos "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
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

	rolePermissionRepo := repositories.NewBunRolePermissionRepository(ctx.DB)
	userAccessRepo := repositories.NewBunUserAccessRepository(ctx.DB)
	impersonationRepo := repositories.NewBunImpersonationRepository(ctx.DB)
	userStateRepo := repositories.NewBunUserStateRepository(ctx.DB)
	sessionStateRepo := repositories.NewBunSessionStateRepository(ctx.DB)

	coreUserRepo := coreinternalrepos.NewBunUserRepository(ctx.DB)

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(rootservices.SessionService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceSession.String())
	}

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceToken.String())
	}

	adminUseCases := usecases.NewAdminUseCases(
		p.config,
		rolePermissionRepo,
		userAccessRepo,
		impersonationRepo,
		userStateRepo,
		sessionStateRepo,
		coreUserRepo,
		sessionService,
		tokenService,
		ctx.GetConfig().Session.ExpiresIn,
	)
	p.Api = NewAPI(
		adminUseCases,
		rolePermissionRepo,
		userAccessRepo,
		impersonationRepo,
		userStateRepo,
		sessionStateRepo,
	)
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
