package oauth2

import (
	"context"
	"errors"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

type API struct {
	UseCases *usecases.UseCases
}

func BuildAPI(plugin *OAuth2Plugin) *API {
	return &API{UseCases: plugin.useCases}
}

func BuildUseCases(p *OAuth2Plugin) *usecases.UseCases {
	globalConfig := p.globalConfig
	trustedOrigins := globalConfig.Security.TrustedOrigins

	useCases := &usecases.UseCases{
		AuthorizeUseCase: usecases.NewAuthorizeUseCase(
			p.providerRegistry,
			p.logger,
			trustedOrigins,
			p.flowStateRepo,
			p.tokenService,
			p.pluginConfig.FlowStateTTL,
		),
		CallbackUseCase: usecases.NewCallbackUseCase(
			p.globalConfig,
			p.providerRegistry,
			p.logger,
			p.flowStateRepo,
			p.userService,
			p.accountService,
			p.sessionService,
			p.tokenService,
		),
	}

	if p.clientService != nil && p.clientCredentialsIssuer != nil {
		useCases.ClientCredentialsUseCase = usecases.NewClientCredentialsUseCase(
			p.clientService,
			p.clientCredentialsIssuer,
			p.pluginConfig.AllowedScopes,
			p.pluginConfig.ClientCredentialsTokenTTL,
			p.logger,
		)
	}

	return useCases
}

// Authorize initiates an OAuth2 authorization flow
func (a *API) Authorize(ctx context.Context, req *types.AuthorizeRequest) (*usecases.AuthorizeResult, error) {
	return a.UseCases.AuthorizeUseCase.Authorize(ctx, req)
}

// Callback handles the OAuth2 callback
func (a *API) Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error) {
	return a.UseCases.CallbackUseCase.Callback(ctx, req, ipAddress, userAgent)
}

// IssueClientCredentialsToken issues a machine token for a confidential OAuth
// client using the client credentials grant.
func (a *API) IssueClientCredentialsToken(ctx context.Context, req *types.TokenRequest) (*types.TokenResult, error) {
	if a.UseCases == nil || a.UseCases.ClientCredentialsUseCase == nil {
		return nil, errors.New("client credentials grant is not enabled")
	}
	return a.UseCases.ClientCredentialsUseCase.IssueToken(ctx, req)
}
