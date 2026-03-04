package admin

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type API struct {
	useCases           *usecases.AdminUseCases
	rolePermissionRepo repositories.RolePermissionRepository
	userAccessRepo     repositories.UserAccessRepository
	impersonationRepo  repositories.ImpersonationRepository
	userStateRepo      repositories.UserStateRepository
	sessionStateRepo   repositories.SessionStateRepository
}

func NewAPI(
	useCases *usecases.AdminUseCases,
	rolePermissionRepo repositories.RolePermissionRepository,
	userAccessRepo repositories.UserAccessRepository,
	impersonationRepo repositories.ImpersonationRepository,
	userStateRepo repositories.UserStateRepository,
	sessionStateRepo repositories.SessionStateRepository,
) *API {
	return &API{
		useCases:           useCases,
		rolePermissionRepo: rolePermissionRepo,
		userAccessRepo:     userAccessRepo,
		impersonationRepo:  impersonationRepo,
		userStateRepo:      userStateRepo,
		sessionStateRepo:   sessionStateRepo,
	}
}

func (a *API) RolePermissionRepository() repositories.RolePermissionRepository {
	return a.rolePermissionRepo
}

func (a *API) UserAccessRepository() repositories.UserAccessRepository {
	return a.userAccessRepo
}

func (a *API) ImpersonationRepository() repositories.ImpersonationRepository {
	return a.impersonationRepo
}

func (a *API) UserStateRepository() repositories.UserStateRepository {
	return a.userStateRepo
}

func (a *API) SessionStateRepository() repositories.SessionStateRepository {
	return a.sessionStateRepo
}

// User management

func (a *API) GetAllUsers(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	return a.useCases.GetAllUsers(ctx, cursor, limit)
}

func (a *API) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return a.useCases.GetUserByID(ctx, userID)
}

func (a *API) CreateUser(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	return a.useCases.CreateUser(ctx, request)
}

func (a *API) UpdateUser(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return a.useCases.UpdateUser(ctx, userID, request)
}

func (a *API) DeleteUser(ctx context.Context, userID string) error {
	return a.useCases.DeleteUser(ctx, userID)
}

// Roles and permissions

func (a *API) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return a.useCases.GetAllRoles(ctx)
}

func (a *API) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return a.useCases.GetRoleByID(ctx, roleID)
}

func (a *API) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return a.useCases.CreateRole(ctx, req)
}

func (a *API) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return a.useCases.UpdateRole(ctx, roleID, req)
}

func (a *API) DeleteRole(ctx context.Context, roleID string) error {
	return a.useCases.DeleteRole(ctx, roleID)
}

func (a *API) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return a.useCases.GetAllPermissions(ctx)
}

func (a *API) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return a.useCases.CreatePermission(ctx, req)
}

func (a *API) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return a.useCases.UpdatePermission(ctx, permissionID, req)
}

func (a *API) DeletePermission(ctx context.Context, permissionID string) error {
	return a.useCases.DeletePermission(ctx, permissionID)
}

func (a *API) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return a.useCases.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (a *API) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return a.useCases.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (a *API) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return a.useCases.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

// User roles and permissions

func (a *API) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return a.useCases.GetUserRoles(ctx, userID)
}

func (a *API) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return a.useCases.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (a *API) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return a.useCases.RemoveRoleFromUser(ctx, userID, roleID)
}

func (a *API) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return a.useCases.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

func (a *API) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return a.useCases.GetUserEffectivePermissions(ctx, userID)
}

// User access and permissions

func (a *API) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	return a.useCases.HasPermissions(ctx, userID, requiredPermissions)
}

func (a *API) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return a.useCases.GetUserWithRolesByID(ctx, userID)
}

func (a *API) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return a.useCases.GetUserWithPermissionsByID(ctx, userID)
}

// Impersonation

func (a *API) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return a.useCases.GetAllImpersonations(ctx)
}

func (a *API) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return a.useCases.GetImpersonationByID(ctx, impersonationID)
}

func (a *API) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return a.useCases.StartImpersonation(ctx, actorUserID, actorSessionID, req)
}

func (a *API) StopImpersonation(ctx context.Context, actorUserID string, req types.StopImpersonationRequest) error {
	return a.useCases.StopImpersonation(ctx, actorUserID, req)
}

// User state

func (a *API) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return a.useCases.GetUserState(ctx, userID)
}

func (a *API) UpsertUserState(ctx context.Context, userID string, req types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.UpsertUserState(ctx, userID, req, actorUserID)
}

func (a *API) DeleteUserState(ctx context.Context, userID string) error {
	return a.useCases.DeleteUserState(ctx, userID)
}

func (a *API) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return a.useCases.GetBannedUserStates(ctx)
}

func (a *API) BanUser(ctx context.Context, userID string, req types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.BanUser(ctx, userID, req, actorUserID)
}

func (a *API) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return a.useCases.UnbanUser(ctx, userID)
}

// Session state

func (a *API) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return a.useCases.GetSessionState(ctx, sessionID)
}

func (a *API) UpsertSessionState(ctx context.Context, sessionID string, req types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.UpsertSessionState(ctx, sessionID, req, actorUserID)
}

func (a *API) DeleteSessionState(ctx context.Context, sessionID string) error {
	return a.useCases.DeleteSessionState(ctx, sessionID)
}

func (a *API) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.RevokeSession(ctx, sessionID, reason, actorUserID)
}

func (a *API) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	return a.useCases.GetUserAdminSessions(ctx, userID)
}

func (a *API) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return a.useCases.GetRevokedSessionStates(ctx)
}
