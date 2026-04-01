package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/types"
)

type UseCases struct {
	roles           *RolesUseCase
	permissions     *PermissionsUseCase
	rolePermissions *RolePermissionsUseCase
	userRoles       *UserRolesUseCase
	userPermissions *UserPermissionsUseCase
}

func NewAccessControlUseCases(
	roles *RolesUseCase,
	permissions *PermissionsUseCase,
	rolePermissions *RolePermissionsUseCase,
	userRoles *UserRolesUseCase,
	userPermissions *UserPermissionsUseCase,
) *UseCases {
	return &UseCases{
		roles:           roles,
		permissions:     permissions,
		rolePermissions: rolePermissions,
		userRoles:       userRoles,
		userPermissions: userPermissions,
	}
}

func (u *UseCases) RolesUseCase() *RolesUseCase {
	return u.roles
}

func (u *UseCases) PermissionsUseCase() *PermissionsUseCase {
	return u.permissions
}

func (u *UseCases) RolePermissionsUseCase() *RolePermissionsUseCase {
	return u.rolePermissions
}

func (u *UseCases) UserRolesUseCase() *UserRolesUseCase {
	return u.userRoles
}

func (u *UseCases) UserPermissionsUseCase() *UserPermissionsUseCase {
	return u.userPermissions
}

// Roles

func (u *UseCases) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return u.roles.CreateRole(ctx, req)
}

func (u *UseCases) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return u.roles.GetAllRoles(ctx)
}

func (u *UseCases) GetRoleByName(ctx context.Context, roleName string) (*types.Role, error) {
	return u.roles.GetRoleByName(ctx, roleName)
}

func (u *UseCases) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return u.roles.GetRoleByID(ctx, roleID)
}

func (u *UseCases) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.roles.UpdateRole(ctx, roleID, req)
}

func (u *UseCases) DeleteRole(ctx context.Context, roleID string) error {
	return u.roles.DeleteRole(ctx, roleID)
}

// Permissions

func (u *UseCases) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.permissions.CreatePermission(ctx, req)
}

func (u *UseCases) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return u.permissions.GetAllPermissions(ctx)
}

func (u *UseCases) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	return u.permissions.GetPermissionByID(ctx, permissionID)
}

func (u *UseCases) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.permissions.UpdatePermission(ctx, permissionID, req)
}

func (u *UseCases) DeletePermission(ctx context.Context, permissionID string) error {
	return u.permissions.DeletePermission(ctx, permissionID)
}

// Role Permissions

func (u *UseCases) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	return u.rolePermissions.GetRolePermissions(ctx, roleID)
}

func (u *UseCases) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return u.rolePermissions.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (u *UseCases) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return u.rolePermissions.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (u *UseCases) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return u.rolePermissions.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

// User Roles

func (u *UseCases) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return u.userRoles.GetUserRoles(ctx, userID)
}

func (u *UseCases) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.userRoles.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

func (u *UseCases) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.userRoles.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (u *UseCases) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return u.userRoles.RemoveRoleFromUser(ctx, userID, roleID)
}

// User Permissions

func (u *UseCases) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return u.userPermissions.GetUserPermissions(ctx, userID)
}

func (u *UseCases) HasPermissions(ctx context.Context, userID string, permissionKeys []string) (bool, error) {
	return u.userPermissions.HasPermissions(ctx, userID, permissionKeys)
}
