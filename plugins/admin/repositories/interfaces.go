package repositories

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type RolePermissionRepository interface {
	CreateRole(ctx context.Context, role *types.Role) error
	GetAllRoles(ctx context.Context) ([]types.Role, error)
	GetRoleByID(ctx context.Context, roleID string) (*types.Role, error)
	UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error)
	DeleteRole(ctx context.Context, roleID string) (bool, error)
	CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error)

	GetAllPermissions(ctx context.Context) ([]types.Permission, error)
	GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error)
	CreatePermission(ctx context.Context, permission *types.Permission) error
	UpdatePermissionDescription(ctx context.Context, permissionID string, description *string) (bool, error)
	DeletePermission(ctx context.Context, permissionID string) (bool, error)
	CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error)

	GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error)
	ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error
	AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error
	RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error

	ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error
	AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error
	RemoveUserRole(ctx context.Context, userID string, roleID string) error
}

type UserAccessRepository interface {
	GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error)
	GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error)
	GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error)
	GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error)
}

type ImpersonationRepository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	CreateImpersonation(ctx context.Context, impersonation *types.Impersonation) error
	GetActiveImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error)
	GetLatestActiveImpersonationByActor(ctx context.Context, actorUserID string) (*types.Impersonation, error)
	EndImpersonation(ctx context.Context, impersonationID string, endedByUserID *string) error
	GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error)
	GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error)
}

type UserStateRepository interface {
	GetByUserID(ctx context.Context, userID string) (*types.AdminUserState, error)
	Upsert(ctx context.Context, state *types.AdminUserState) error
	Delete(ctx context.Context, userID string) error
	GetBanned(ctx context.Context) ([]types.AdminUserState, error)
}

type SessionStateRepository interface {
	GetBySessionID(ctx context.Context, sessionID string) (*types.AdminSessionState, error)
	Upsert(ctx context.Context, state *types.AdminSessionState) error
	Delete(ctx context.Context, sessionID string) error
	GetRevoked(ctx context.Context) ([]types.AdminSessionState, error)
	SessionExists(ctx context.Context, sessionID string) (bool, error)
	GetByUserID(ctx context.Context, userID string) ([]types.AdminUserSession, error)
}
