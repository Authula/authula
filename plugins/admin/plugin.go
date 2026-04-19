package admin

import (
	"fmt"

	coreinternalrepos "github.com/Authula/authula/internal/repositories"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
	rootservices "github.com/Authula/authula/services"
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
	return Routes(p.Api)
}

func (p *AdminPlugin) Close() error {
	return nil
}
