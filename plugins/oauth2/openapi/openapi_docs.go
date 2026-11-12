package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/oauth2/types"
)

// callbackFormRequest documents the form-encoded body Apple posts back. It is
// separate from types.CallbackRequest because swaggest builds a form body only
// from `form` tags, and the shared type carries path and query parameters too.
type callbackFormRequest struct {
	Provider string `path:"provider" required:"true"`
	Code     string `form:"code" required:"true"`
	State    string `form:"state" required:"true"`
	Error    string `form:"error"`
	User     string `form:"user"`
}

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
			"/oauth2/callback/{provider}",
			openapi.WithOperationID("oauthCallbackFormPost"),
			openapi.WithSummary("OAuth2 callback (form post)"),
			openapi.WithDescription("Handles providers that return the authorization response as a cross-site form POST, such as Sign in with Apple."),
			openapi.WithTags("OAuth2 Plugin"),
			openapi.WithRequest(&callbackFormRequest{}, openapi.WithFormContentType()),
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
