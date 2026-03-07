package admin

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminhandlers "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/handlers"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type routeUseCases struct {
	users         usecases.UsersUseCase
	impersonation usecases.ImpersonationUseCase
	state         usecases.StateUseCase
}

func newRouteUseCases(api *API) routeUseCases {
	return routeUseCases{
		users:         api.useCases.UsersUseCase(),
		impersonation: api.useCases.ImpersonationUseCase(),
		state:         api.useCases.StateUseCase(),
	}
}

func Routes(api *API) []models.Route {
	usecases := newRouteUseCases(api)

	return []models.Route{
		// User management
		{Method: http.MethodPost, Path: "/admin/users", Handler: adminhandlers.NewCreateUserHandler(usecases.users).Handler()},
		{Method: http.MethodGet, Path: "/admin/users", Handler: adminhandlers.NewGetAllUsersHandler(usecases.users).Handler()},
		{Method: http.MethodGet, Path: "/admin/users/{user_id}", Handler: adminhandlers.NewGetUserByIDHandler(usecases.users).Handler()},
		{Method: http.MethodPatch, Path: "/admin/users/{user_id}", Handler: adminhandlers.NewUpdateUserHandler(usecases.users).Handler()},
		{Method: http.MethodDelete, Path: "/admin/users/{user_id}", Handler: adminhandlers.NewDeleteUserHandler(usecases.users).Handler()},

		// User state
		{Method: http.MethodGet, Path: "/admin/users/{user_id}/state", Handler: adminhandlers.NewGetUserStateHandler(usecases.state).Handler()},
		{Method: http.MethodPost, Path: "/admin/users/{user_id}/state", Handler: adminhandlers.NewUpsertUserStateHandler(usecases.state).Handler()},
		{Method: http.MethodDelete, Path: "/admin/users/{user_id}/state", Handler: adminhandlers.NewDeleteUserStateHandler(usecases.state).Handler()},
		{Method: http.MethodGet, Path: "/admin/users/states/banned", Handler: adminhandlers.NewGetBannedUserStatesHandler(usecases.state).Handler()},
		{Method: http.MethodPost, Path: "/admin/users/{user_id}/ban", Handler: adminhandlers.NewBanUserHandler(usecases.state).Handler()},
		{Method: http.MethodPost, Path: "/admin/users/{user_id}/unban", Handler: adminhandlers.NewUnbanUserHandler(usecases.state).Handler()},
		{Method: http.MethodGet, Path: "/admin/users/{user_id}/sessions", Handler: adminhandlers.NewGetUserAdminSessionsHandler(usecases.state).Handler()},

		// Session state
		{Method: http.MethodGet, Path: "/admin/sessions/{session_id}/state", Handler: adminhandlers.NewGetSessionStateHandler(usecases.state).Handler()},
		{Method: http.MethodPost, Path: "/admin/sessions/{session_id}/state", Handler: adminhandlers.NewUpsertSessionStateHandler(usecases.state).Handler()},
		{Method: http.MethodDelete, Path: "/admin/sessions/{session_id}/state", Handler: adminhandlers.NewDeleteSessionStateHandler(usecases.state).Handler()},
		{Method: http.MethodGet, Path: "/admin/sessions/states/revoked", Handler: adminhandlers.NewGetRevokedSessionStatesHandler(usecases.state).Handler()},
		{Method: http.MethodPost, Path: "/admin/sessions/{session_id}/revoke", Handler: adminhandlers.NewRevokeSessionHandler(usecases.state).Handler()},

		// Impersonation
		{Method: http.MethodGet, Path: "/admin/impersonations", Handler: adminhandlers.NewGetAllImpersonationsHandler(usecases.impersonation).Handler()},
		{Method: http.MethodGet, Path: "/admin/impersonations/{impersonation_id}", Handler: adminhandlers.NewGetImpersonationByIDHandler(usecases.impersonation).Handler()},
		{Method: http.MethodPost, Path: "/admin/impersonations", Handler: adminhandlers.NewStartImpersonationHandler(usecases.impersonation).Handler()},
		{Method: http.MethodPost, Path: "/admin/impersonations/{impersonation_id}/end", Handler: adminhandlers.NewStopImpersonationHandler(usecases.impersonation).Handler()},
	}
}
