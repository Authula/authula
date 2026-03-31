package repositories

import (
	"context"
	"time"

	"github.com/Authula/authula/plugins/access-control/types"
)

type RolesRepository interface {
	CreateRole(ctx context.Context, role *types.Role) error
	GetAllRoles(ctx context.Context) ([]types.Role, error)
	GetRoleByID(ctx context.Context, roleID string) (*types.Role, error)
	GetRoleByName(ctx context.Context, roleName string) (*types.Role, error)
	UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error)
	DeleteRole(ctx context.Context, roleID string) (bool, error)
}

type PermissionsRepository interface {
	GetAllPermissions(ctx context.Context) ([]types.Permission, error)
	GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error)
	GetPermissionByKey(ctx context.Context, permissionKey string) (*types.Permission, error)
	CreatePermission(ctx context.Context, permission *types.Permission) error
	UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error)
	DeletePermission(ctx context.Context, permissionID string) (bool, error)
}

type RolePermissionsRepository interface {
	GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error)
	ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error
	AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error
	RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error
	CountRolesByPermission(ctx context.Context, permissionID string) (int, error)
}

type UserRolesRepository interface {
	GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error)
	ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error
	AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error
	RemoveUserRole(ctx context.Context, userID string, roleID string) error
	CountUsersByRole(ctx context.Context, roleID string) (int, error)
}

type UserPermissionsRepository interface {
	GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error)
	HasPermissions(ctx context.Context, userID string, permissionKeys []string) (bool, error)
}
