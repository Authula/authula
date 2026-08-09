package accesscontrol

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type API struct {
	rolesService           *services.RolesService
	permissionsService     *services.PermissionsService
	rolePermissionsService *services.RolePermissionsService
	userRolesService       *services.UserRolesService
	userPermissionsService *services.UserPermissionsService
}

func NewAPI(
	rolesService *services.RolesService,
	permissionsService *services.PermissionsService,
	rolePermissionsService *services.RolePermissionsService,
	userRolesService *services.UserRolesService,
	userPermissionsService *services.UserPermissionsService,
) *API {
	return &API{
		rolesService:           rolesService,
		permissionsService:     permissionsService,
		rolePermissionsService: rolePermissionsService,
		userRolesService:       userRolesService,
		userPermissionsService: userPermissionsService,
	}
}

// Roles

func (a *API) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return a.rolesService.GetAllRoles(ctx)
}

func (a *API) GetRoleByName(ctx context.Context, roleName string) (*types.Role, error) {
	return a.rolesService.GetRoleByName(ctx, roleName)
}

func (a *API) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return a.rolesService.GetRoleByID(ctx, roleID)
}

func (a *API) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return a.rolesService.CreateRole(ctx, req)
}

func (a *API) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return a.rolesService.UpdateRole(ctx, roleID, req)
}

func (a *API) DeleteRole(ctx context.Context, roleID string) error {
	return a.rolesService.DeleteRole(ctx, roleID)
}

// Permissions

func (a *API) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return a.permissionsService.CreatePermission(ctx, req)
}

func (a *API) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return a.permissionsService.GetAllPermissions(ctx)
}

func (a *API) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	return a.permissionsService.GetPermissionByID(ctx, permissionID)
}

func (a *API) GetPermissionByKey(ctx context.Context, permissionKey string) (*types.Permission, error) {
	return a.permissionsService.GetPermissionByKey(ctx, permissionKey)
}

func (a *API) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	return a.rolePermissionsService.GetRolePermissions(ctx, roleID)
}

func (a *API) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return a.permissionsService.UpdatePermission(ctx, permissionID, req)
}

func (a *API) DeletePermission(ctx context.Context, permissionID string) error {
	return a.permissionsService.DeletePermission(ctx, permissionID)
}

// Role permissions

func (a *API) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return a.rolePermissionsService.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (a *API) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return a.rolePermissionsService.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (a *API) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return a.rolePermissionsService.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

// User roles

func (a *API) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return a.userRolesService.GetUserRoles(ctx, userID)
}

func (a *API) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return a.userRolesService.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (a *API) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return a.userRolesService.RemoveRoleFromUser(ctx, userID, roleID)
}

func (a *API) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return a.userRolesService.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

// User permissions

func (a *API) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return a.userPermissionsService.GetUserPermissions(ctx, userID)
}

func (a *API) HasPermissions(ctx context.Context, userID string, permissionNames []string) (bool, error) {
	return a.userPermissionsService.HasPermissions(ctx, userID, permissionNames)
}
