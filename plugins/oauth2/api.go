package oauth2

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/usecases"
)

type API struct {
	UseCases *usecases.UseCases
}

func BuildAPI(plugin *OAuth2Plugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{UseCases: useCases}
}

func BuildUseCases(p *OAuth2Plugin) *usecases.UseCases {
	globalConfig := p.globalConfig
	trustedOrigins := globalConfig.Security.TrustedOrigins

	return &usecases.UseCases{
		AuthorizeUseCase: usecases.NewAuthorizeUseCase(
			p.providerRegistry,
			p.logger,
			trustedOrigins,
			p.hmacKey,
		),
		CallbackUseCase: usecases.NewCallbackUseCase(
			p.globalConfig,
			p.providerRegistry,
			p.logger,
			p.hmacKey,
			p.userService,
			p.accountService,
			p.sessionService,
			p.tokenService,
		),
	}
}

// Authorize initiates an OAuth2 authorization flow
func (a *API) Authorize(ctx context.Context, req *types.AuthorizeRequest) (*usecases.AuthorizeResult, error) {
	return a.UseCases.AuthorizeUseCase.Authorize(ctx, req)
}

// Callback handles the OAuth2 callback
func (a *API) Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error) {
	return a.UseCases.CallbackUseCase.Callback(ctx, req, ipAddress, userAgent)
}
