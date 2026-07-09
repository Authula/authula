package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodGet,
			"/oauth2/authorize/{provider}",
			openapi.WithOperationID("oauthAuthorize"),
			openapi.WithSummary("Authorize with OAuth2 provider"),
			openapi.WithDescription("Initiates the OAuth2 authorization flow with the specified provider. Returns the provider's authorization URL to redirect the user to."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&types.AuthorizeRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.AuthorizeResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/oauth2/callback/{provider}",
			openapi.WithOperationID("oauthCallback"),
			openapi.WithSummary("OAuth2 callback"),
			openapi.WithDescription("Handles the OAuth2 callback from the provider. Exchanges the authorization code for tokens, creates or links a user account, and returns the authenticated user and session."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&types.CallbackRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.CallbackResponse{}),
		),
	)
}
