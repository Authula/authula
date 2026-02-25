package admin

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminhandlers "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/handlers"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type routeUseCases struct {
	users          usecases.UsersUseCase
	rolePermission usecases.RolePermissionUseCase
	userAccess     usecases.UserRolesUseCase
	impersonation  usecases.ImpersonationUseCase
	state          usecases.StateUseCase
}

func newRouteUseCases(api *API) routeUseCases {
	return routeUseCases{
		users:          api.useCases.UsersUseCase(),
		rolePermission: api.useCases.RolePermissionUseCase(),
		userAccess:     api.useCases.UserAccessUseCase(),
		impersonation:  api.useCases.ImpersonationUseCase(),
		state:          api.useCases.StateUseCase(),
	}
}

func Routes(api *API) []models.Route {
	usecases := newRouteUseCases(api)

	return []models.Route{
		// User management
		{Method: http.MethodGet, Path: "/admin/users", Handler: adminhandlers.NewGetAllUsersHandler(usecases.users).Handler()},
		{Method: http.MethodPost, Path: "/admin/users", Handler: adminhandlers.NewCreateUserHandler(usecases.users).Handler()},
		{Method: http.MethodGet, Path: "/admin/users/{user_id}", Handler: adminhandlers.NewGetUserByIDHandler(usecases.users).Handler()},
		{Method: http.MethodPatch, Path: "/admin/users/{user_id}", Handler: adminhandlers.NewUpdateUserHandler(usecases.users).Handler()},
		{Method: http.MethodDelete, Path: "/admin/users/{user_id}", Handler: adminhandlers.NewDeleteUserHandler(usecases.users).Handler()},

		// Roles and permissions
		{Method: http.MethodGet, Path: "/admin/roles", Handler: adminhandlers.NewGetAllRolesHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPost, Path: "/admin/roles", Handler: adminhandlers.NewCreateRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/admin/roles/{role_id}", Handler: adminhandlers.NewGetRoleByIDHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPatch, Path: "/admin/roles/{role_id}", Handler: adminhandlers.NewUpdateRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/admin/roles/{role_id}", Handler: adminhandlers.NewDeleteRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/admin/permissions", Handler: adminhandlers.NewGetAllPermissionsHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPost, Path: "/admin/permissions", Handler: adminhandlers.NewCreatePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPatch, Path: "/admin/permissions/{permission_id}", Handler: adminhandlers.NewUpdatePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/admin/permissions/{permission_id}", Handler: adminhandlers.NewDeletePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPost, Path: "/admin/roles/{role_id}/permissions", Handler: adminhandlers.NewAddRolePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPut, Path: "/admin/roles/{role_id}/permissions", Handler: adminhandlers.NewReplaceRolePermissionsHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/admin/roles/{role_id}/permissions/{permission_id}", Handler: adminhandlers.NewRemoveRolePermissionHandler(usecases.rolePermission).Handler()},

		// User roles and permissions
		{Method: http.MethodGet, Path: "/admin/users/{user_id}/roles", Handler: adminhandlers.NewGetUserRolesHandler(usecases.userAccess).Handler()},
		{Method: http.MethodPost, Path: "/admin/users/{user_id}/roles", Handler: adminhandlers.NewAssignUserRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPut, Path: "/admin/users/{user_id}/roles", Handler: adminhandlers.NewReplaceUserRolesHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/admin/users/{user_id}/roles/{role_id}", Handler: adminhandlers.NewRemoveUserRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/admin/users/{user_id}/permissions", Handler: adminhandlers.NewGetUserEffectivePermissionsHandler(usecases.userAccess).Handler()},

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
		{Method: http.MethodPost, Path: "/admin/impersonations/{impersonation_id}/end", Handler: adminhandlers.NewEndImpersonationByIDHandler(usecases.impersonation).Handler()},
	}
}
