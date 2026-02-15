package magiclink

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	types "github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type MagicLinkPlugin struct {
	globalConfig        *models.Config
	pluginConfig        *types.MagicLinkPluginConfig
	logger              models.Logger
	ctx                 *models.PluginContext
	userService         rootservices.UserService
	accountService      rootservices.AccountService
	sessionService      rootservices.SessionService
	verificationService rootservices.VerificationService
	tokenService        rootservices.TokenService
	mailerService       rootservices.MailerService
	Api                 *API
}

func New(config types.MagicLinkPluginConfig) *MagicLinkPlugin {
	config.ApplyDefaults()
	return &MagicLinkPlugin{
		pluginConfig: &config,
	}
}

func (p *MagicLinkPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginMagicLink.String(),
		Version:     "1.0.0",
		Description: "Magic link plugin for passwordless authentication.",
	}
}

func (p *MagicLinkPlugin) Config() any {
	return p.pluginConfig
}

func (p *MagicLinkPlugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx
	p.globalConfig = ctx.GetConfig()

	if err := util.LoadPluginConfig(p.globalConfig, p.Metadata().ID, p.pluginConfig); err != nil {
		p.logger.Warn("failed to load magic link plugin config, using defaults", map[string]any{
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

	mailerService, ok := ctx.ServiceRegistry.Get(models.ServiceMailer.String()).(rootservices.MailerService)
	if !ok {
		return fmt.Errorf("mailer service not available in service registry")
	}
	p.mailerService = mailerService

	p.Api = BuildAPI(p)

	return nil
}

func (p *MagicLinkPlugin) Close() error {
	return nil
}

func (p *MagicLinkPlugin) Routes() []models.Route {
	return Routes(p)
}
