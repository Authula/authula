package repositories

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type RolePermissionRepository interface {
	CreateRole(ctx context.Context, role *types.Role) error
	GetAllRoles(ctx context.Context) ([]types.Role, error)
	GetRoleByID(ctx context.Context, roleID string) (*types.Role, error)
	UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error)
	DeleteRole(ctx context.Context, roleID string) (bool, error)
	GetAllPermissions(ctx context.Context) ([]types.Permission, error)
	GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error)
	CreatePermission(ctx context.Context, permission *types.Permission) error
	UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error)
	DeletePermission(ctx context.Context, permissionID string) (bool, error)
	GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error)
	ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error
	AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error
	RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error
	ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error
	AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error
	RemoveUserRole(ctx context.Context, userID string, roleID string) error
	CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error)
	CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error)
}

type UserAccessRepository interface {
	GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error)
	GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error)
	GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error)
	GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error)
}
