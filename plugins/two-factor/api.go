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
			p.userService,
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.totpService,
			p.backupCodeService,
			p.twoFactorRepo,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		Disable: usecases.NewDisableUseCase(
			p.accountService,
			p.passwordService,
			p.twoFactorRepo,
			p.ctx.EventBus,
			p.logger,
		),
		GetTOTPURI: usecases.NewGetTOTPURIUseCase(
			p.userService,
			p.accountService,
			p.passwordService,
			p.tokenService,
			p.totpService,
			p.twoFactorRepo,
			p.pluginConfig,
		),
		VerifyTOTP: usecases.NewVerifyTOTPUseCase(
			p.tokenService,
			p.sessionService,
			p.userService,
			p.verificationService,
			p.totpService,
			p.twoFactorRepo,
			p.globalConfig,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		GenerateBackupCodes: usecases.NewGenerateBackupCodesUseCase(
			p.accountService,
			p.passwordService,
			p.backupCodeService,
			p.twoFactorRepo,
		),
		VerifyBackupCode: usecases.NewVerifyBackupCodeUseCase(
			p.tokenService,
			p.sessionService,
			p.userService,
			p.verificationService,
			p.backupCodeService,
			p.twoFactorRepo,
			p.globalConfig,
			p.pluginConfig,
			p.ctx.EventBus,
			p.logger,
		),
		ViewBackupCodes: usecases.NewViewBackupCodesUseCase(
			p.accountService,
			p.passwordService,
			p.twoFactorRepo,
		),
	}
}
