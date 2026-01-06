package core

import (
	"context"
	"embed"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/repositories"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/security"
	coreservices "github.com/GoBetterAuth/go-better-auth/plugins/core/services"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/types"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type CorePlugin struct {
	config              types.CorePluginConfig
	logger              models.Logger
	ctx                 *models.PluginContext
	userService         services.UserService
	accountService      services.AccountService
	sessionService      services.SessionService
	verificationService services.VerificationService
	tokenService        services.TokenService
	db                  bun.IDB
	signer              security.TokenSigner
	Api                 *API
}

func New(config types.CorePluginConfig) *CorePlugin {
	return &CorePlugin{
		config: config,
	}
}

func (p *CorePlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginCore.String(),
		Version:     "1.0.0",
		Description: "Core authentication plugin providing users, accounts, sessions, and verification.",
	}
}

func (p *CorePlugin) Config() any {
	return p.config
}

func (p *CorePlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx
	p.db = ctx.DB
	globalConfig := ctx.GetConfig()

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	signer, err := security.NewHMACSigner(globalConfig.Secret)
	if err != nil {
		return err
	}
	p.signer = signer

	userRepo := repositories.NewBunUserRepository(p.db)
	accountRepo := repositories.NewBunAccountRepository(p.db)
	sessionRepo := repositories.NewBunSessionRepository(p.db)
	verificationRepo := repositories.NewBunVerificationRepository(p.db)
	tokenRepo := repositories.NewCryptoTokenRepository(globalConfig.Secret)

	p.userService = coreservices.NewUserService(userRepo, p.config.DatabaseHooks)
	p.accountService = coreservices.NewAccountService(globalConfig, accountRepo, tokenRepo, p.config.DatabaseHooks)
	p.sessionService = coreservices.NewSessionService(sessionRepo, signer, p.config.DatabaseHooks)
	p.verificationService = coreservices.NewVerificationService(verificationRepo, signer, p.config.DatabaseHooks)
	p.tokenService = coreservices.NewTokenService(tokenRepo)

	ctx.ServiceRegistry.Register(models.ServiceUser.String(), p.userService)
	ctx.ServiceRegistry.Register(models.ServiceAccount.String(), p.accountService)
	ctx.ServiceRegistry.Register(models.ServiceSession.String(), p.sessionService)
	ctx.ServiceRegistry.Register(models.ServiceVerification.String(), p.verificationService)
	ctx.ServiceRegistry.Register(models.ServiceToken.String(), p.tokenService)

	p.Api = BuildAPI(p)

	return nil
}

func (p *CorePlugin) Migrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	return GetMigrations(ctx, dbProvider)
}

func (p *CorePlugin) Routes() []models.Route {
	return Routes(p)
}

func (p *CorePlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(p.ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	// Secret rotation support
	if config.Secret != "" {
		signer, err := security.NewHMACSigner(config.Secret)
		if err != nil {
			return err
		}
		p.signer = signer
	}

	return nil
}

func (p *CorePlugin) Close() error {
	p.logger.Info("core plugin shutting down")
	return nil
}
