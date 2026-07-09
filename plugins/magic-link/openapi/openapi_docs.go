package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/magic-link/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodPost,
			"/magic-link/sign-in",
			openapi.WithOperationID("signInWithMagicLink"),
			openapi.WithSummary("Sign in with magic link"),
			openapi.WithDescription("Sends a magic link to the given email address if an account exists. Optionally creates a new account if sign-up is enabled."),
			openapi.WithTags("Magic Link Plugin"),
			openapi.WithRequest(&types.MagicLinkSignInRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.MagicLinkSignInResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/magic-link/verify",
			openapi.WithOperationID("verifyMagicLink"),
			openapi.WithSummary("Verify magic link token"),
			openapi.WithDescription("Verifies a magic link token. If a callback_url is provided and trusted, redirects the user with an exchange token appended. Otherwise returns the exchange token in the JSON response."),
			openapi.WithTags("Magic Link Plugin"),
			openapi.WithRequest(&types.MagicLinkVerifyRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.MagicLinkVerifyResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/magic-link/exchange",
			openapi.WithOperationID("exchangeMagicLink"),
			openapi.WithSummary("Exchange magic link token for session"),
			openapi.WithDescription("Exchanges a verified magic link token for an authenticated user session."),
			openapi.WithTags("Magic Link Plugin"),
			openapi.WithRequest(&types.MagicLinkExchangeRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.MagicLinkExchangeResponse{}),
		),
	)
}
