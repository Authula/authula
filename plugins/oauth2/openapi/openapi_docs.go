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
			openapi.WithDescription("Initiates the OAuth2 authorization flow. Returns the provider authorization URL and sets short-lived, signed flow cookies for state, redirect, and PKCE."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&types.AuthorizeRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.AuthorizeResponse{}),
			openapi.WithResponseStatus(http.StatusNotFound, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusUnprocessableEntity, &types.ErrorResponse{}),
			openapi.WithResponseStatus(http.StatusInternalServerError, &types.ErrorResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/oauth2/callback/{provider}",
			openapi.WithOperationID("oauthCallback"),
			openapi.WithSummary("OAuth2 callback"),
			openapi.WithDescription("Handles the OAuth2 callback. Validates the provider-bound state, consumes PKCE, exchanges the code, links the provider account, and returns the user and a safe session view. When a validated redirect_to was supplied, responds with 302 and a Location header."),
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
	)
}
