package accesscontrol

import (
	"net/http"

	"github.com/Authula/authula/middleware"
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

func newRouteUseCases(plugin *AccessControlPlugin) routeUseCases {
	return routeUseCases{
		roles:           plugin.useCases.RolesUseCase(),
		permissions:     plugin.useCases.PermissionsUseCase(),
		rolePermissions: plugin.useCases.RolePermissionsUseCase(),
		userRoles:       plugin.useCases.UserRolesUseCase(),
		userPermissions: plugin.useCases.UserPermissionsUseCase(),
	}
}

func Routes(plugin *AccessControlPlugin) []models.Route {
	usecases := newRouteUseCases(plugin)

	return []models.Route{
		// Roles
		{
			Method: http.MethodPost,
			Path:   "/access-control/roles",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewCreateRoleHandler(usecases.roles).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/roles",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetAllRolesHandler(usecases.roles).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/roles/by-name/{role_name}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetRoleByNameHandler(usecases.roles).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/roles/{role_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetRoleByIDHandler(usecases.roles).Handler(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/access-control/roles/{role_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewUpdateRoleHandler(usecases.roles).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/access-control/roles/{role_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewDeleteRoleHandler(usecases.roles).Handler(),
		},

		// Permissions
		{
			Method: http.MethodPost,
			Path:   "/access-control/permissions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewCreatePermissionHandler(usecases.permissions).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/permissions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetAllPermissionsHandler(usecases.permissions).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/permissions/by-key/{permission_key}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetPermissionByKeyHandler(usecases.permissions).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/permissions/{permission_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetPermissionByIDHandler(usecases.permissions).Handler(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/access-control/permissions/{permission_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewUpdatePermissionHandler(usecases.permissions).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/access-control/permissions/{permission_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewDeletePermissionHandler(usecases.permissions).Handler(),
		},

		// Role permissions
		{
			Method: http.MethodPost,
			Path:   "/access-control/roles/{role_id}/permissions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewAddRolePermissionHandler(usecases.rolePermissions).Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/access-control/roles/{role_id}/permissions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetRolePermissionsHandler(usecases.rolePermissions).Handler(),
		},
		{
			Method: http.MethodPut,
			Path:   "/access-control/roles/{role_id}/permissions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewReplaceRolePermissionsHandler(usecases.rolePermissions).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/access-control/roles/{role_id}/permissions/{permission_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewRemoveRolePermissionHandler(usecases.rolePermissions).Handler(),
		},

		// User roles
		{
			Method: http.MethodGet,
			Path:   "/access-control/users/{user_id}/roles",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetUserRolesHandler(usecases.userRoles).Handler(),
		},
		{
			Method: http.MethodPut,
			Path:   "/access-control/users/{user_id}/roles",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewReplaceUserRolesHandler(usecases.userRoles).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/access-control/users/{user_id}/roles",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewAssignUserRoleHandler(usecases.userRoles).Handler(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/access-control/users/{user_id}/roles/{role_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewRemoveUserRoleHandler(usecases.userRoles).Handler(),
		},

		// User permissions
		{
			Method: http.MethodGet,
			Path:   "/access-control/users/{user_id}/permissions",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewGetUserPermissionsHandler(usecases.userPermissions).Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/access-control/users/{user_id}/permissions/check",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: handlers.NewCheckUserPermissionsHandler(usecases.userPermissions).Handler(),
		},
	}
}
