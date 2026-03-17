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

// UseCases returns the use cases for external access.
func (a *API) UseCases() *usecases.UseCases {
	return a.useCases
}

func BuildUseCases(p *TOTPPlugin) *usecases.UseCases {
	return &usecases.UseCases{
		Enable: usecases.NewEnableUseCase(
			p.userService,
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.verificationService,
			p.totpService,
			p.backupCodeService,
			p.totpRepo,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		Disable: usecases.NewDisableUseCase(
			p.accountService,
			p.passwordService,
			p.totpRepo,
			p.ctx.EventBus,
			p.logger,
		),
		GetTOTPURI: usecases.NewGetTOTPURIUseCase(
			p.userService,
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.totpService,
			p.totpRepo,
			p.pluginConfig,
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
			p.accountService,
			p.passwordService,
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
		ViewBackupCodes: usecases.NewViewBackupCodesUseCase(
			p.accountService,
			p.passwordService,
			p.totpRepo,
		),
	}
}
