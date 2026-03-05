package twofactor

import (
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func BuildAPI(plugin *TwoFactorPlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

// UseCases returns the use cases for external access.
func (a *API) UseCases() *usecases.UseCases {
	return a.useCases
}

func BuildUseCases(p *TwoFactorPlugin) *usecases.UseCases {
	return &usecases.UseCases{
		Enable: usecases.NewEnableUseCase(
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.totpService,
			p.backupCodeService,
			p.repo,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		Disable: usecases.NewDisableUseCase(
			p.accountService,
			p.passwordService,
			p.repo,
			p.ctx.EventBus,
			p.logger,
		),
		GetTOTPURI: usecases.NewGetTOTPURIUseCase(
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.totpService,
			p.repo,
			p.pluginConfig,
		),
		VerifyTOTP: usecases.NewVerifyTOTPUseCase(
			p.tokenService,
			p.sessionService,
			p.userService,
			p.totpService,
			p.repo,
			p.globalConfig,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		GenerateBackupCodes: usecases.NewGenerateBackupCodesUseCase(
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.backupCodeService,
			p.repo,
		),
		VerifyBackupCode: usecases.NewVerifyBackupCodeUseCase(
			p.tokenService,
			p.sessionService,
			p.userService,
			p.backupCodeService,
			p.repo,
			p.globalConfig,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		ViewBackupCodes: usecases.NewViewBackupCodesUseCase(
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.repo,
		),
	}
}
