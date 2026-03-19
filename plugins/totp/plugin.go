package totp

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/systems"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type TOTPPlugin struct {
	globalConfig        *models.Config
	pluginConfig        *types.TOTPPluginConfig
	logger              models.Logger
	ctx                 *models.PluginContext
	userService         rootservices.UserService
	accountService      rootservices.AccountService
	sessionService      rootservices.SessionService
	verificationService rootservices.VerificationService
	tokenService        rootservices.TokenService
	passwordService     rootservices.PasswordService
	totpService         *services.TOTPService
	backupCodeService   *services.BackupCodeService
	totpRepo            *repository.TOTPRepository
	cleanupSystem       *systems.TrustedDevicesCleanupSystem
	Api                 *API
}

func New(config types.TOTPPluginConfig) *TOTPPlugin {
	config.ApplyDefaults()
	return &TOTPPlugin{
		pluginConfig: &config,
	}
}

func (p *TOTPPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginTOTP.String(),
		Version:     "1.0.0",
		Description: "TOTP plugin with backup codes for two-factor authentication.",
	}
}

func (p *TOTPPlugin) Config() any {
	return p.pluginConfig
}

func (p *TOTPPlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx
	p.globalConfig = ctx.GetConfig()

	if err := util.LoadPluginConfig(p.globalConfig, p.Metadata().ID, p.pluginConfig); err != nil {
		p.logger.Warn("failed to load totp plugin config, using defaults", map[string]any{
			"error": err.Error(),
		})
	}

	userService, ok := ctx.ServiceRegistry.Get(models.ServiceUser.String()).(rootservices.UserService)
	if !ok {
		return fmt.Errorf("user service not available in service registry")
	}
	p.userService = userService

	accountService, ok := ctx.ServiceRegistry.Get(models.ServiceAccount.String()).(rootservices.AccountService)
	if !ok {
		return fmt.Errorf("account service not available in service registry")
	}
	p.accountService = accountService

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(rootservices.SessionService)
	if !ok {
		return fmt.Errorf("session service not available in service registry")
	}
	p.sessionService = sessionService

	verificationService, ok := ctx.ServiceRegistry.Get(models.ServiceVerification.String()).(rootservices.VerificationService)
	if !ok {
		return fmt.Errorf("verification service not available in service registry")
	}
	p.verificationService = verificationService

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("token service not available in service registry")
	}
	p.tokenService = tokenService

	passwordService, ok := ctx.ServiceRegistry.Get(models.ServicePassword.String()).(rootservices.PasswordService)
	if !ok {
		return fmt.Errorf("password service not available in service registry")
	}
	p.passwordService = passwordService

	p.totpService = &services.TOTPService{
		Digits:        6,
		PeriodSeconds: 30,
	}
	p.backupCodeService = &services.BackupCodeService{
		Count:           p.pluginConfig.BackupCodeCount,
		PasswordService: p.passwordService,
	}

	p.totpRepo = repository.NewTOTPRepository(ctx.DB)

	p.cleanupSystem = systems.NewTrustedDevicesCleanupSystem(
		p.logger,
		p.pluginConfig,
		p.totpRepo,
	)

	if err := p.cleanupSystem.Init(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize trusted devices cleanup system: %w", err)
	}

	p.Api = BuildAPI(p)

	return nil
}

func (p *TOTPPlugin) Close() error {
	return p.cleanupSystem.Close()
}

func (p *TOTPPlugin) Routes() []models.Route {
	return Routes(p)
}

func (p *TOTPPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *TOTPPlugin) Migrations(provider string) []migrations.Migration {
	return totpMigrationsForProvider(provider)
}

func (p *TOTPPlugin) DependsOn() []string {
	return nil
}
