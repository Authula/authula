package usecases

import (
	"context"
	"time"

	corerepositories "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type AdminUseCases struct {
	rolePermission RolePermissionUseCase
	userAccess     UserRolesUseCase
	users          UsersUseCase
	impersonation  ImpersonationUseCase
	state          StateUseCase
}

func NewAdminUseCases(
	config types.AdminPluginConfig,
	rolePermissionRepo repositories.RolePermissionRepository,
	userAccessRepo repositories.UserAccessRepository,
	impersonationRepo repositories.ImpersonationRepository,
	userStateRepo repositories.UserStateRepository,
	sessionStateRepo repositories.SessionStateRepository,
	userRepo corerepositories.UserRepository,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
	sessionExpiresIn time.Duration,
) *AdminUseCases {
	return &AdminUseCases{
		rolePermission: NewRolePermissionUseCase(rolePermissionRepo),
		userAccess:     NewUserRolesUseCase(userAccessRepo),
		users:          NewUsersUseCase(userRepo),
		impersonation: NewImpersonationUseCase(
			impersonationRepo,
			sessionStateRepo,
			sessionService,
			tokenService,
			sessionExpiresIn,
			config.ImpersonationMaxExpiresIn,
		),
		state: NewStateUseCase(userStateRepo, sessionStateRepo, impersonationRepo),
	}
}

func (u *AdminUseCases) RolePermissionUseCase() RolePermissionUseCase {
	return u.rolePermission
}

func (u *AdminUseCases) UserAccessUseCase() UserRolesUseCase {
	return u.userAccess
}

func (u *AdminUseCases) UsersUseCase() UsersUseCase {
	return u.users
}

func (u *AdminUseCases) ImpersonationUseCase() ImpersonationUseCase {
	return u.impersonation
}

func (u *AdminUseCases) StateUseCase() StateUseCase {
	return u.state
}

func (u *AdminUseCases) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.rolePermission.CreatePermission(ctx, req)
}

func (u *AdminUseCases) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return u.rolePermission.GetAllPermissions(ctx)
}

func (u *AdminUseCases) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.rolePermission.UpdatePermission(ctx, permissionID, req)
}

func (u *AdminUseCases) DeletePermission(ctx context.Context, permissionID string) error {
	return u.rolePermission.DeletePermission(ctx, permissionID)
}

func (u *AdminUseCases) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return u.rolePermission.CreateRole(ctx, req)
}

func (u *AdminUseCases) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return u.rolePermission.GetAllRoles(ctx)
}

func (u *AdminUseCases) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return u.rolePermission.GetRoleByID(ctx, roleID)
}

func (u *AdminUseCases) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.rolePermission.UpdateRole(ctx, roleID, req)
}

func (u *AdminUseCases) DeleteRole(ctx context.Context, roleID string) error {
	return u.rolePermission.DeleteRole(ctx, roleID)
}

func (u *AdminUseCases) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return u.rolePermission.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (u *AdminUseCases) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return u.rolePermission.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (u *AdminUseCases) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return u.rolePermission.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

func (u *AdminUseCases) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.rolePermission.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

func (u *AdminUseCases) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.rolePermission.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (u *AdminUseCases) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return u.rolePermission.RemoveRoleFromUser(ctx, userID, roleID)
}

func (u *AdminUseCases) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return u.userAccess.GetUserRoles(ctx, userID)
}

func (u *AdminUseCases) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return u.userAccess.GetUserEffectivePermissions(ctx, userID)
}

func (u *AdminUseCases) GetAllUsers(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	return u.users.GetAll(ctx, cursor, limit)
}

func (u *AdminUseCases) CreateUser(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	return u.users.Create(ctx, request)
}

func (u *AdminUseCases) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return u.users.GetByID(ctx, userID)
}

func (u *AdminUseCases) UpdateUser(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return u.users.Update(ctx, userID, request)
}

func (u *AdminUseCases) DeleteUser(ctx context.Context, userID string) error {
	return u.users.Delete(ctx, userID)
}

func (u *AdminUseCases) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	return u.userAccess.HasPermissions(ctx, userID, requiredPermissions)
}

func (u *AdminUseCases) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return u.userAccess.GetUserWithRolesByID(ctx, userID)
}

func (u *AdminUseCases) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return u.userAccess.GetUserWithPermissionsByID(ctx, userID)
}

func (u *AdminUseCases) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	return u.userAccess.GetUserAuthorizationProfile(ctx, userID)
}

func (u *AdminUseCases) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return u.impersonation.StartImpersonation(ctx, actorUserID, actorSessionID, req)
}

func (u *AdminUseCases) StopImpersonation(ctx context.Context, actorUserID string, request types.StopImpersonationRequest) error {
	return u.impersonation.StopImpersonation(ctx, actorUserID, request)
}

func (u *AdminUseCases) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return u.impersonation.GetAllImpersonations(ctx)
}

func (u *AdminUseCases) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return u.impersonation.GetImpersonationByID(ctx, impersonationID)
}

func (u *AdminUseCases) UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.UpsertUserState(ctx, userID, request, actorUserID)
}

func (u *AdminUseCases) BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.BanUser(ctx, userID, request, actorUserID)
}

func (u *AdminUseCases) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return u.state.UnbanUser(ctx, userID)
}

func (u *AdminUseCases) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return u.state.GetUserState(ctx, userID)
}

func (u *AdminUseCases) DeleteUserState(ctx context.Context, userID string) error {
	return u.state.DeleteUserState(ctx, userID)
}

func (u *AdminUseCases) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return u.state.GetBannedUserStates(ctx)
}

func (u *AdminUseCases) UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.UpsertSessionState(ctx, sessionID, request, actorUserID)
}

func (u *AdminUseCases) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.RevokeSession(ctx, sessionID, reason, actorUserID)
}

func (u *AdminUseCases) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	return u.state.GetUserAdminSessions(ctx, userID)
}

func (u *AdminUseCases) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return u.state.GetSessionState(ctx, sessionID)
}

func (u *AdminUseCases) DeleteSessionState(ctx context.Context, sessionID string) error {
	return u.state.DeleteSessionState(ctx, sessionID)
}

func (u *AdminUseCases) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return u.state.GetRevokedSessionStates(ctx)
}
