package openapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Authula/authula/internal/types"
	"github.com/Authula/authula/openapi"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService, basePath string) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodGet,
			fmt.Sprintf("%s/me", basePath),
			openapi.WithOperationID("getMe"),
			openapi.WithSummary("Get current user"),
			openapi.WithDescription("Retrieves the authenticated user's profile and current session."),
			openapi.WithTags("Core"),
			openapi.WithResponseStatus(http.StatusOK, &types.GetMeResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			fmt.Sprintf("%s/sign-out", basePath),
			openapi.WithOperationID("signOut"),
			openapi.WithSummary("Sign out"),
			openapi.WithDescription("Signs out the authenticated user. Optionally sign out a specific session or all sessions."),
			openapi.WithTags("Core"),
			openapi.WithRequest(&types.SignOutRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.SignOutResponse{}),
		),
	)
}
