package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/jwt/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodPost,
			"/token/refresh",
			openapi.WithOperationID("refreshToken"),
			openapi.WithSummary("Refresh JWT token"),
			openapi.WithDescription("Exchanges a valid refresh token for a new access token and refresh token pair."),
			openapi.WithTags("JWT Plugin"),
			openapi.WithRequest(&types.RefreshTokenRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.RefreshTokenResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/.well-known/jwks.json",
			openapi.WithOperationID("getJWKS"),
			openapi.WithSummary("Get JWKS"),
			openapi.WithDescription("Returns the JSON Web Key Set (JWKS) containing the public keys used to verify JWT signatures."),
			openapi.WithTags("JWT Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.WellKnownJWKSResponse{}),
		),
	)
}
