package totp

import (
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func BuildAPI(plugin *TOTPPlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

func (a *API) UseCases() *usecases.UseCases {
	return a.useCases
}

func BuildUseCases(p *TOTPPlugin) *usecases.UseCases {
	return &usecases.UseCases{
		Enable: usecases.NewEnableUseCase(
			p.pluginConfig,
			p.logger,
			p.ctx.EventBus,
			p.userService,
			p.tokenService,
			p.verificationService,
			p.totpService,
			p.backupCodeService,
			p.totpRepo,
		),
		Disable: usecases.NewDisableUseCase(
			p.logger,
			p.ctx.EventBus,
			p.totpRepo,
		),
		GetTOTPURI: usecases.NewGetTOTPURIUseCase(
			p.pluginConfig,
			p.userService,
			p.tokenService,
			p.totpService,
			p.totpRepo,
		),
		VerifyTOTP: usecases.NewVerifyTOTPUseCase(
			p.globalConfig,
			p.pluginConfig,
			p.logger,
			p.ctx.EventBus,
			p.tokenService,
			p.sessionService,
			p.userService,
			p.verificationService,
			p.totpService,
			p.totpRepo,
		),
		GenerateBackupCodes: usecases.NewGenerateBackupCodesUseCase(
			p.backupCodeService,
			p.totpRepo,
		),
		VerifyBackupCode: usecases.NewVerifyBackupCodeUseCase(
			p.globalConfig,
			p.pluginConfig,
			p.logger,
			p.ctx.EventBus,
			p.tokenService,
			p.sessionService,
			p.userService,
			p.verificationService,
			p.backupCodeService,
			p.totpRepo,
		),
	}
}
