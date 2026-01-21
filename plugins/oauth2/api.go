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
			int(p.pluginConfig.CookieTTL.Seconds()),
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
		RefreshUseCase: usecases.NewRefreshUseCase(
			p.providerRegistry,
			p.logger,
		),
		LinkAccountUseCase: usecases.NewLinkAccountUseCase(
			p.providerRegistry,
			p.logger,
			p.userService,
			p.accountService,
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

// Refresh refreshes an OAuth2 token for an authenticated user
func (a *API) Refresh(ctx context.Context, userID, providerID string) (*usecases.RefreshResult, error) {
	return a.UseCases.RefreshUseCase.Refresh(ctx, userID, providerID)
}

// LinkAccount links an OAuth2 account to an existing user
func (a *API) LinkAccount(ctx context.Context, userID, providerID, providerAccountID string) (*usecases.LinkAccountResult, error) {
	return a.UseCases.LinkAccountUseCase.LinkAccount(ctx, userID, providerID, providerAccountID)
}
