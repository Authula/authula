package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/core/types"
	"github.com/Authula/authula/openapi"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodGet,
			"/me",
			openapi.WithOperationID("getMe"),
			openapi.WithSummary("Get current user"),
			openapi.WithDescription("Retrieves the authenticated user's profile and current session."),
			openapi.WithTags("Core"),
			openapi.WithResponseStatus(http.StatusOK, &types.GetMeResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/sign-out",
			openapi.WithOperationID("signOut"),
			openapi.WithSummary("Sign out"),
			openapi.WithDescription("Signs out the authenticated user. With no body, revokes the session that authenticated the request (400 if the request carries no session id). Pass session_id to revoke one of your own sessions (404 if it does not exist, 403 if it belongs to another user), or sign_out_all to revoke every session."),
			openapi.WithTags("Core"),
			openapi.WithRequest(&types.SignOutRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.SignOutResponse{}),
		),
	)
}
