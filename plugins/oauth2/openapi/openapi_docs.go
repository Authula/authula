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
			openapi.WithDescription("Initiates the OAuth2 authorization flow."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&types.AuthorizeRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.AuthorizeResponse{}),
			openapi.WithResponseStatus(http.StatusNotFound, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusUnprocessableEntity, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusTooManyRequests, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusInternalServerError, &types.ErrorResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/oauth2/callback/{provider}",
			openapi.WithOperationID("oauthCallback"),
			openapi.WithSummary("OAuth2 callback"),
			openapi.WithDescription("Handles the OAuth2 callback."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&types.CallbackRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.CallbackResponse{}),
			openapi.WithResponseStatus(http.StatusFound, &types.RedirectResponse{}),
			openapi.WithResponseStatus(http.StatusBadRequest, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusUnauthorized, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusNotFound, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusConflict, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusUnprocessableEntity, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusBadGateway, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusInternalServerError, &types.ErrorResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/oauth2/token",
			openapi.WithOperationID("oauthToken"),
			openapi.WithSummary("OAuth2 client credentials token"),
			openapi.WithDescription("Authenticates a confidential OAuth client and returns a Bearer access token using the client credentials grant."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&types.TokenEndpointRequest{}, openapi.WithFormContentType()),
			openapi.WithResponseStatus(http.StatusOK, &types.TokenResponse{}),
			openapi.WithResponseStatus(http.StatusBadRequest, &types.TokenErrorResponse{}),
			openapi.WithResponseStatus(http.StatusUnauthorized, &types.TokenErrorResponse{}),
			openapi.WithResponseStatus(http.StatusInternalServerError, &types.ErrorResponse{}),
		),
	)
}
