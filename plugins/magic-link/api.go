package magiclink

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func (a *API) SignIn(
	ctx context.Context,
	name *string,
	email string,
	callbackURL *string,
) (*types.SignInResult, error) {
	return a.useCases.SignInUseCase.SignIn(ctx, name, email, callbackURL)
}

func (a *API) Verify(
	ctx context.Context,
	token string,
	ipAddress *string,
	userAgent *string,
) (string, error) {
	return a.useCases.VerifyUseCase.Verify(ctx, token, ipAddress, userAgent)
}

func (a *API) Exchange(
	ctx context.Context,
	token string,
	ipAddress *string,
	userAgent *string,
) (*types.ExchangeResult, error) {
	return a.useCases.ExchangeUseCase.Exchange(ctx, token, ipAddress, userAgent)
}

func BuildAPI(plugin *MagicLinkPlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

func BuildUseCases(p *MagicLinkPlugin) *usecases.UseCases {
	return &usecases.UseCases{
		SignInUseCase: &usecases.SignInUseCaseImpl{
			GlobalConfig:        p.globalConfig,
			PluginConfig:        p.pluginConfig,
			Logger:              p.logger,
			UserService:         p.userService,
			AccountService:      p.accountService,
			VerificationService: p.verificationService,
			TokenService:        p.tokenService,
			MailerService:       p.mailerService,
		},
		VerifyUseCase: &usecases.VerifyUseCaseImpl{
			GlobalConfig:        p.globalConfig,
			PluginConfig:        p.pluginConfig,
			Logger:              p.logger,
			UserService:         p.userService,
			VerificationService: p.verificationService,
			TokenService:        p.tokenService,
		},
		ExchangeUseCase: &usecases.ExchangeUseCaseImpl{
			GlobalConfig:        p.globalConfig,
			PluginConfig:        p.pluginConfig,
			Logger:              p.logger,
			UserService:         p.userService,
			AccountService:      p.accountService,
			SessionService:      p.sessionService,
			VerificationService: p.verificationService,
			TokenService:        p.tokenService,
		},
	}
}
