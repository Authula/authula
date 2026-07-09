package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/internal/types"
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
			openapi.WithDescription("Signs out the authenticated user. Optionally sign out a specific session or all sessions."),
			openapi.WithTags("Core"),
			openapi.WithRequest(&types.SignOutRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.SignOutResponse{}),
		),
	)
}
