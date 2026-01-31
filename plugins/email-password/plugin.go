package email_password

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type EmailPasswordPlugin struct {
	globalConfig        *models.Config
	pluginConfig        types.EmailPasswordPluginConfig
	logger              models.Logger
	ctx                 *models.PluginContext
	db                  bun.IDB
	userService         rootservices.UserService
	accountService      rootservices.AccountService
	sessionService      rootservices.SessionService
	verificationService rootservices.VerificationService
	tokenService        rootservices.TokenService
	passwordService     rootservices.PasswordService
	mailerService       rootservices.MailerService
	Api                 *API
}

func New(config types.EmailPasswordPluginConfig) *EmailPasswordPlugin {
	config.ApplyDefaults()
	return &EmailPasswordPlugin{
		pluginConfig: config,
	}
}

func (p *EmailPasswordPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginEmailPassword.String(),
		Version:     "1.0.0",
		Description: "Email/Password authentication plugin with password hashing, verification, and reset flows.",
	}
}

func (p *EmailPasswordPlugin) Config() any {
	return p.pluginConfig
}

func (p *EmailPasswordPlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx
	p.db = ctx.DB
	globalConfig := ctx.GetConfig()
	p.globalConfig = globalConfig

	if err := util.LoadPluginConfig(globalConfig, p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
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

	p.passwordService = services.NewArgon2PasswordService()
	ctx.ServiceRegistry.Register(models.ServicePassword.String(), p.passwordService)

	mailerService, ok := ctx.ServiceRegistry.Get(models.ServiceMailer.String()).(rootservices.MailerService)
	if !ok {
		return fmt.Errorf("mailer service not available in service registry")
	}
	p.mailerService = mailerService

	p.Api = BuildAPI(p)

	return nil
}

func (p *EmailPasswordPlugin) Routes() []models.Route {
	return Routes(p)
}

func (p *EmailPasswordPlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(p.ctx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
	}
	return nil
}

func (p *EmailPasswordPlugin) Close() error {
	return nil
}
