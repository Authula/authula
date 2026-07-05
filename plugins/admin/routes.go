package admin

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	adminhandlers "github.com/Authula/authula/plugins/admin/handlers"
	"github.com/Authula/authula/plugins/admin/usecases"
)

type routeUseCases struct {
	users         usecases.UsersUseCase
	accounts      usecases.AccountsUseCase
	state         usecases.StateUseCase
	impersonation usecases.ImpersonationUseCase
}

func newRouteUseCases(api *API) routeUseCases {
	return routeUseCases{
		users:         api.useCases.UsersUseCase(),
		accounts:      api.useCases.AccountsUseCase(),
		state:         api.useCases.StateUseCase(),
		impersonation: api.useCases.ImpersonationUseCase(),
	}
}

func (p *AdminPlugin) buildRoutes(api *API) []models.Route {
	usecases := newRouteUseCases(api)

	return []models.Route{
		// User management
		{
			Method: http.MethodPost,
			Path:   "/admin/users",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewCreateUserHandler(usecases.users).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/users",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetAllUsersHandler(usecases.users).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/users/{user_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetUserByIDHandler(usecases.users).Handler(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/admin/users/{user_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewUpdateUserHandler(usecases.users).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/admin/users/{user_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewDeleteUserHandler(usecases.users).Handler(),
		},

		// Account management
		{
			Method: http.MethodPost,
			Path:   "/admin/users/{user_id}/accounts",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewCreateAccountHandler(usecases.accounts).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/users/{user_id}/accounts",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetUserAccountsHandler(usecases.accounts).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/accounts/{id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetAccountByIDHandler(usecases.accounts).Handler(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/admin/accounts/{id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewUpdateAccountHandler(usecases.accounts).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/admin/accounts/{id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewDeleteAccountHandler(usecases.accounts).Handler(),
		},

		// User state
		{
			Method: http.MethodGet,
			Path:   "/admin/users/{user_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetUserStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/users/{user_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewCreateUserStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/admin/users/{user_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewUpdateUserStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/admin/users/{user_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewDeleteUserStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/users/states/banned",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetBannedUserStatesHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/users/{user_id}/ban",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewBanUserHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/users/{user_id}/unban",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewUnbanUserHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/users/{user_id}/sessions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetUserAdminSessionsHandler(usecases.state).Handler(),
		},

		// Session state
		{
			Method: http.MethodGet,
			Path:   "/admin/sessions/{session_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetSessionStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/sessions/{session_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewCreateSessionStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/admin/sessions/{session_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewUpdateSessionStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/admin/sessions/{session_id}/state",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewDeleteSessionStateHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/sessions/states/revoked",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetRevokedSessionStatesHandler(usecases.state).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/sessions/{session_id}/revoke",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewRevokeSessionHandler(usecases.state).Handler(),
		},

		// Impersonation
		{
			Method: http.MethodGet,
			Path:   "/admin/impersonations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetAllImpersonationsHandler(usecases.impersonation).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/admin/impersonations/{impersonation_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: adminhandlers.NewGetImpersonationByIDHandler(usecases.impersonation).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/impersonations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: adminhandlers.NewStartImpersonationHandler(
				usecases.impersonation,
				p.globalConfig,
			).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/admin/impersonations/{impersonation_id}/stop",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: adminhandlers.NewStopImpersonationHandler(
				usecases.impersonation,
				p.globalConfig,
			).Handler(),
		},
	}
}
