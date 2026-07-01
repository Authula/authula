package usecases

import (
	"context"

	"github.com/Authula/authula/models"
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

func (u *UseCases) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	return u.roles.CreateRole(ctx, actor, req)
}

func (u *UseCases) GetAllRoles(ctx context.Context, actor *models.Actor) ([]types.Role, error) {
	return u.roles.GetAllRoles(ctx, actor)
}

func (u *UseCases) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	return u.roles.GetRoleByName(ctx, actor, roleName)
}

func (u *UseCases) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	return u.roles.GetRoleByID(ctx, actor, roleID)
}

func (u *UseCases) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.roles.UpdateRole(ctx, actor, roleID, req)
}

func (u *UseCases) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	return u.roles.DeleteRole(ctx, actor, roleID)
}

// Permissions

func (u *UseCases) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.permissions.CreatePermission(ctx, actor, req)
}

func (u *UseCases) GetAllPermissions(ctx context.Context, actor *models.Actor) ([]types.Permission, error) {
	return u.permissions.GetAllPermissions(ctx, actor)
}

func (u *UseCases) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	return u.permissions.GetPermissionByID(ctx, actor, permissionID)
}

func (u *UseCases) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.permissions.UpdatePermission(ctx, actor, permissionID, req)
}

func (u *UseCases) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	return u.permissions.DeletePermission(ctx, actor, permissionID)
}

// Role Permissions

func (u *UseCases) GetRolePermissions(ctx context.Context, actor *models.Actor, roleID string) ([]types.UserPermissionInfo, error) {
	return u.rolePermissions.GetRolePermissions(ctx, actor, roleID)
}

func (u *UseCases) AddPermissionToRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string, grantedByUserID *string) error {
	return u.rolePermissions.AddPermissionToRole(ctx, actor, roleID, permissionID, grantedByUserID)
}

func (u *UseCases) RemovePermissionFromRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string) error {
	return u.rolePermissions.RemovePermissionFromRole(ctx, actor, roleID, permissionID)
}

func (u *UseCases) ReplaceRolePermissions(ctx context.Context, actor *models.Actor, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return u.rolePermissions.ReplaceRolePermissions(ctx, actor, roleID, permissionIDs, grantedByUserID)
}

// User Roles

func (u *UseCases) GetUserRoles(ctx context.Context, actor *models.Actor, userID string) ([]types.UserRoleInfo, error) {
	return u.userRoles.GetUserRoles(ctx, actor, userID)
}

func (u *UseCases) ReplaceUserRoles(ctx context.Context, actor *models.Actor, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.userRoles.ReplaceUserRoles(ctx, actor, userID, roleIDs, assignedByUserID)
}

func (u *UseCases) AssignRoleToUser(ctx context.Context, actor *models.Actor, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.userRoles.AssignRoleToUser(ctx, actor, userID, req, assignedByUserID)
}

func (u *UseCases) RemoveRoleFromUser(ctx context.Context, actor *models.Actor, userID string, roleID string) error {
	return u.userRoles.RemoveRoleFromUser(ctx, actor, userID, roleID)
}

// User Permissions

func (u *UseCases) GetSelfUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	return u.userPermissions.GetSelfUserPermissions(ctx, actor, userID)
}

func (u *UseCases) GetUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	return u.userPermissions.GetUserPermissions(ctx, actor, userID)
}

func (u *UseCases) HasPermissions(ctx context.Context, actor *models.Actor, userID string, permissionKeys []string) (bool, error) {
	return u.userPermissions.HasPermissions(ctx, actor, userID, permissionKeys)
}
