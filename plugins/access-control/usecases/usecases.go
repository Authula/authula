package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type UseCases struct {
	rolePermission RolePermissionUseCase
	userAccess     UserRolesUseCase
}

func NewAccessControlUseCases(rolePermission RolePermissionUseCase, userAccess UserRolesUseCase) *UseCases {
	return &UseCases{
		rolePermission: rolePermission,
		userAccess:     userAccess,
	}
}

func (u *UseCases) RolePermissionUseCase() RolePermissionUseCase {
	return u.rolePermission
}

func (u *UseCases) UserAccessUseCase() UserRolesUseCase {
	return u.userAccess
}

func (u *UseCases) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.rolePermission.CreatePermission(ctx, req)
}

func (u *UseCases) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return u.rolePermission.GetAllPermissions(ctx)
}

func (u *UseCases) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	return u.rolePermission.GetRolePermissions(ctx, roleID)
}

func (u *UseCases) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.rolePermission.UpdatePermission(ctx, permissionID, req)
}

func (u *UseCases) DeletePermission(ctx context.Context, permissionID string) error {
	return u.rolePermission.DeletePermission(ctx, permissionID)
}

func (u *UseCases) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return u.rolePermission.CreateRole(ctx, req)
}

func (u *UseCases) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return u.rolePermission.GetAllRoles(ctx)
}

func (u *UseCases) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return u.rolePermission.GetRoleByID(ctx, roleID)
}

func (u *UseCases) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.rolePermission.UpdateRole(ctx, roleID, req)
}

func (u *UseCases) DeleteRole(ctx context.Context, roleID string) error {
	return u.rolePermission.DeleteRole(ctx, roleID)
}

func (u *UseCases) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return u.rolePermission.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (u *UseCases) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return u.rolePermission.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (u *UseCases) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return u.rolePermission.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

func (u *UseCases) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.rolePermission.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

func (u *UseCases) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.rolePermission.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (u *UseCases) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return u.rolePermission.RemoveRoleFromUser(ctx, userID, roleID)
}

func (u *UseCases) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return u.userAccess.GetUserRoles(ctx, userID)
}

func (u *UseCases) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return u.userAccess.GetUserEffectivePermissions(ctx, userID)
}

func (u *UseCases) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	return u.userAccess.HasPermissions(ctx, userID, requiredPermissions)
}

func (u *UseCases) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return u.userAccess.GetUserWithRolesByID(ctx, userID)
}

func (u *UseCases) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return u.userAccess.GetUserWithPermissionsByID(ctx, userID)
}

func (u *UseCases) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	return u.userAccess.GetUserAuthorizationProfile(ctx, userID)
}
