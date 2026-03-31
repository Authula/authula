package accesscontrol

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/handlers"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type routeUseCases struct {
	roles           *usecases.RolesUseCase
	permissions     *usecases.PermissionsUseCase
	rolePermissions *usecases.RolePermissionsUseCase
	userRoles       *usecases.UserRolesUseCase
	userPermissions *usecases.UserPermissionsUseCase
}

func newRouteUseCases(api *API) routeUseCases {
	return routeUseCases{
		roles:           api.useCases.RolesUseCase(),
		permissions:     api.useCases.PermissionsUseCase(),
		rolePermissions: api.useCases.RolePermissionsUseCase(),
		userRoles:       api.useCases.UserRolesUseCase(),
		userPermissions: api.useCases.UserPermissionsUseCase(),
	}
}

func Routes(api *API) []models.Route {
	usecases := newRouteUseCases(api)

	return []models.Route{
		// Roles
		{Method: http.MethodPost, Path: "/access-control/roles", Handler: handlers.NewCreateRoleHandler(usecases.roles).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles", Handler: handlers.NewGetAllRolesHandler(usecases.roles).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles/by-name/{role_name}", Handler: handlers.NewGetRoleByNameHandler(usecases.roles).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles/{role_id}", Handler: handlers.NewGetRoleByIDHandler(usecases.roles).Handler()},
		{Method: http.MethodPatch, Path: "/access-control/roles/{role_id}", Handler: handlers.NewUpdateRoleHandler(usecases.roles).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/roles/{role_id}", Handler: handlers.NewDeleteRoleHandler(usecases.roles).Handler()},

		// Permissions
		{Method: http.MethodPost, Path: "/access-control/permissions", Handler: handlers.NewCreatePermissionHandler(usecases.permissions).Handler()},
		{Method: http.MethodGet, Path: "/access-control/permissions", Handler: handlers.NewGetAllPermissionsHandler(usecases.permissions).Handler()},
		{Method: http.MethodGet, Path: "/access-control/permissions/{permission_id}", Handler: handlers.NewGetPermissionByIDHandler(usecases.permissions).Handler()},
		{Method: http.MethodPatch, Path: "/access-control/permissions/{permission_id}", Handler: handlers.NewUpdatePermissionHandler(usecases.permissions).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/permissions/{permission_id}", Handler: handlers.NewDeletePermissionHandler(usecases.permissions).Handler()},

		// Role permissions
		{Method: http.MethodPost, Path: "/access-control/roles/{role_id}/permissions", Handler: handlers.NewAddRolePermissionHandler(usecases.rolePermissions).Handler()},
		{Method: http.MethodGet, Path: "/access-control/roles/{role_id}/permissions", Handler: handlers.NewGetRolePermissionsHandler(usecases.rolePermissions).Handler()},
		{Method: http.MethodPut, Path: "/access-control/roles/{role_id}/permissions", Handler: handlers.NewReplaceRolePermissionsHandler(usecases.rolePermissions).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/roles/{role_id}/permissions/{permission_id}", Handler: handlers.NewRemoveRolePermissionHandler(usecases.rolePermissions).Handler()},

		// User roles
		{Method: http.MethodGet, Path: "/access-control/users/{user_id}/roles", Handler: handlers.NewGetUserRolesHandler(usecases.userRoles).Handler()},
		{Method: http.MethodPut, Path: "/access-control/users/{user_id}/roles", Handler: handlers.NewReplaceUserRolesHandler(usecases.userRoles).Handler()},
		{Method: http.MethodPost, Path: "/access-control/users/{user_id}/roles", Handler: handlers.NewAssignUserRoleHandler(usecases.userRoles).Handler()},
		{Method: http.MethodDelete, Path: "/access-control/users/{user_id}/roles/{role_id}", Handler: handlers.NewRemoveUserRoleHandler(usecases.userRoles).Handler()},

		// User permissions
		{Method: http.MethodGet, Path: "/access-control/users/{user_id}/permissions", Handler: handlers.NewGetUserPermissionsHandler(usecases.userPermissions).Handler()},
		{Method: http.MethodPost, Path: "/access-control/users/{user_id}/permissions/check", Handler: handlers.NewCheckUserPermissionsHandler(usecases.userPermissions).Handler()},
	}
}
