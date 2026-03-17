package accesscontrol

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/handlers"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/usecases"
)

type routeUseCases struct {
	rolePermission usecases.RolePermissionUseCase
	userAccess     usecases.UserRolesUseCase
}

func newRouteUseCases(api *API) routeUseCases {
	return routeUseCases{
		rolePermission: api.useCases.RolePermissionUseCase(),
		userAccess:     api.useCases.UserAccessUseCase(),
	}
}

func Routes(api *API) []models.Route {
	usecases := newRouteUseCases(api)

	return []models.Route{
		// Roles and permissions
		{Method: http.MethodPost, Path: "/access-control/roles", Handler: handlers.NewCreateRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles", Handler: handlers.NewGetAllRolesHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles/{role_id}", Handler: handlers.NewGetRoleByIDHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPatch, Path: "/access-control/roles/{role_id}", Handler: handlers.NewUpdateRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/roles/{role_id}", Handler: handlers.NewDeleteRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPost, Path: "/access-control/permissions", Handler: handlers.NewCreatePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/access-control/permissions", Handler: handlers.NewGetAllPermissionsHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPatch, Path: "/access-control/permissions/{permission_id}", Handler: handlers.NewUpdatePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/permissions/{permission_id}", Handler: handlers.NewDeletePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPost, Path: "/access-control/roles/{role_id}/permissions", Handler: handlers.NewAddRolePermissionHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles/{role_id}/permissions", Handler: handlers.NewGetRolePermissionsHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPut, Path: "/access-control/roles/{role_id}/permissions", Handler: handlers.NewReplaceRolePermissionsHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/roles/{role_id}/permissions/{permission_id}", Handler: handlers.NewRemoveRolePermissionHandler(usecases.rolePermission).Handler()},

		// User roles and permissions
		{Method: http.MethodGet, Path: "/access-control/users/{user_id}/roles", Handler: handlers.NewGetUserRolesHandler(usecases.userAccess).Handler()},
		{Method: http.MethodPost, Path: "/access-control/users/{user_id}/roles", Handler: handlers.NewAssignUserRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodPut, Path: "/access-control/users/{user_id}/roles", Handler: handlers.NewReplaceUserRolesHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/users/{user_id}/roles/{role_id}", Handler: handlers.NewRemoveUserRoleHandler(usecases.rolePermission).Handler()},
		{Method: http.MethodGet, Path: "/access-control/users/{user_id}/permissions", Handler: handlers.NewGetUserEffectivePermissionsHandler(usecases.userAccess).Handler()},
	}
}
