package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type UsersUseCase interface {
	GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error)
	Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error)
	GetByID(ctx context.Context, userID string) (*models.User, error)
	Update(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error)
	Delete(ctx context.Context, userID string) error
}

type RolePermissionUseCase interface {
	CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error)
	GetAllPermissions(ctx context.Context) ([]types.Permission, error)
	UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error)
	DeletePermission(ctx context.Context, permissionID string) error
	CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error)
	GetAllRoles(ctx context.Context) ([]types.Role, error)
	GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error)
	UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error)
	DeleteRole(ctx context.Context, roleID string) error
	AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error
	RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error
	ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error
	AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error
	RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error
	ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error
}

type UserRolesUseCase interface {
	GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error)
	GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error)
	HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error)
	GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error)
	GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error)
	GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error)
}

type ImpersonationUseCase interface {
	StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error)
	StopImpersonation(ctx context.Context, actorUserID string, request types.StopImpersonationRequest) error
	GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error)
	GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error)
}

type StateUseCase interface {
	UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error)
	BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error)
	UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error)
	GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error)
	DeleteUserState(ctx context.Context, userID string) error
	GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error)
	UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error)
	RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error)
	GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error)
	GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error)
	DeleteSessionState(ctx context.Context, sessionID string) error
	GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error)
}

var _ UsersUseCase = (*usersUseCase)(nil)
var _ RolePermissionUseCase = (*rolePermissionUseCase)(nil)
var _ UserRolesUseCase = (*userRolesUseCase)(nil)
var _ ImpersonationUseCase = (*impersonationUseCase)(nil)
var _ StateUseCase = (*stateUseCase)(nil)
