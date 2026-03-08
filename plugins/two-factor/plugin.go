package twofactor

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type TwoFactorPlugin struct {
	globalConfig        *models.Config
	pluginConfig        *types.TwoFactorPluginConfig
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
	twoFactorRepo       *repository.TwoFactorRepository
	Api                 *API
}

func New(config types.TwoFactorPluginConfig) *TwoFactorPlugin {
	config.ApplyDefaults()
	return &TwoFactorPlugin{
		pluginConfig: &config,
	}
}

func (p *TwoFactorPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginTwoFactor.String(),
		Version:     "1.0.0",
		Description: "Two-factor authentication plugin with TOTP and backup codes.",
	}
}

func (p *TwoFactorPlugin) Config() any {
	return p.pluginConfig
}

func (p *TwoFactorPlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx
	p.globalConfig = ctx.GetConfig()

	if err := util.LoadPluginConfig(p.globalConfig, p.Metadata().ID, p.pluginConfig); err != nil {
		p.logger.Warn("failed to load two-factor plugin config, using defaults", map[string]any{
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

	// Create domain services
	p.totpService = &services.TOTPService{
		Digits:        p.pluginConfig.Digits,
		PeriodSeconds: p.pluginConfig.PeriodSeconds,
	}
	p.backupCodeService = &services.BackupCodeService{
		Count:           p.pluginConfig.BackupCodeCount,
		PasswordService: p.passwordService,
	}

	// Create repository
	p.twoFactorRepo = repository.NewTwoFactorRepository(ctx.DB)

	// Build API
	p.Api = BuildAPI(p)

	return nil
}

func (p *TwoFactorPlugin) Close() error {
	return nil
}

func (p *TwoFactorPlugin) Routes() []models.Route {
	return Routes(p)
}

func (p *TwoFactorPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *TwoFactorPlugin) Migrations(provider string) []migrations.Migration {
	return twoFactorMigrationsForProvider(provider)
}

func (p *TwoFactorPlugin) DependsOn() []string {
	return nil
}
