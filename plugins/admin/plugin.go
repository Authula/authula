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
		Description: "Provides admin operations for users, state, and impersonation.",
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

	impersonationRepo := repositories.NewBunImpersonationRepository(ctx.DB)
	userStateRepo := repositories.NewBunUserStateRepository(ctx.DB)
	sessionStateRepo := repositories.NewBunSessionStateRepository(ctx.DB)

	coreUserRepo := coreinternalrepos.NewBunUserRepository(ctx.DB)
	coreAccountRepo := coreinternalrepos.NewBunAccountRepository(ctx.DB)

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(rootservices.SessionService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceSession.String())
	}

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceToken.String())
	}

	passwordService, ok := ctx.ServiceRegistry.Get(models.ServicePassword.String()).(rootservices.PasswordService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServicePassword.String())
	}

	adminUseCases := usecases.NewAdminUseCases(
		p.config,
		coreUserRepo,
		coreAccountRepo,
		sessionService,
		tokenService,
		passwordService,
		userStateRepo,
		sessionStateRepo,
		impersonationRepo,
		ctx.GetConfig().Session.ExpiresIn,
	)
	p.Api = NewAPI(
		adminUseCases,
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
