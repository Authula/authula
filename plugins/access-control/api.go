package accesscontrol

import (
	"context"

	"github.com/Authula/authula/models"
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

func (a *API) GetAllRoles(ctx context.Context, actor *models.Actor) ([]types.Role, error) {
	return a.rolesService.GetAllRoles(ctx, actor)
}

func (a *API) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	return a.rolesService.GetRoleByName(ctx, actor, roleName)
}

func (a *API) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	return a.rolesService.GetRoleByID(ctx, actor, roleID)
}

func (a *API) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	return a.rolesService.CreateRole(ctx, actor, req)
}

func (a *API) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return a.rolesService.UpdateRole(ctx, actor, roleID, req)
}

func (a *API) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	return a.rolesService.DeleteRole(ctx, actor, roleID)
}

// Permissions

func (a *API) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	return a.permissionsService.CreatePermission(ctx, actor, req)
}

func (a *API) GetAllPermissions(ctx context.Context, actor *models.Actor) ([]types.Permission, error) {
	return a.permissionsService.GetAllPermissions(ctx, actor)
}

func (a *API) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	return a.permissionsService.GetPermissionByID(ctx, actor, permissionID)
}

func (a *API) GetPermissionByKey(ctx context.Context, actor *models.Actor, permissionKey string) (*types.Permission, error) {
	return a.permissionsService.GetPermissionByKey(ctx, actor, permissionKey)
}

func (a *API) GetRolePermissions(ctx context.Context, actor *models.Actor, roleID string) ([]types.UserPermissionInfo, error) {
	return a.rolePermissionsService.GetRolePermissions(ctx, actor, roleID)
}

func (a *API) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return a.permissionsService.UpdatePermission(ctx, actor, permissionID, req)
}

func (a *API) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	return a.permissionsService.DeletePermission(ctx, actor, permissionID)
}

// Role permissions

func (a *API) AddPermissionToRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string, grantedByUserID *string) error {
	return a.rolePermissionsService.AddPermissionToRole(ctx, actor, roleID, permissionID, grantedByUserID)
}

func (a *API) RemovePermissionFromRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string) error {
	return a.rolePermissionsService.RemovePermissionFromRole(ctx, actor, roleID, permissionID)
}

func (a *API) ReplaceRolePermissions(ctx context.Context, actor *models.Actor, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return a.rolePermissionsService.ReplaceRolePermissions(ctx, actor, roleID, permissionIDs, grantedByUserID)
}

// User roles

func (a *API) GetUserRoles(ctx context.Context, actor *models.Actor, userID string) ([]types.UserRoleInfo, error) {
	return a.userRolesService.GetUserRoles(ctx, actor, userID)
}

func (a *API) AssignRoleToUser(ctx context.Context, actor *models.Actor, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return a.userRolesService.AssignRoleToUser(ctx, actor, userID, req, assignedByUserID)
}

func (a *API) RemoveRoleFromUser(ctx context.Context, actor *models.Actor, userID string, roleID string) error {
	return a.userRolesService.RemoveRoleFromUser(ctx, actor, userID, roleID)
}

func (a *API) ReplaceUserRoles(ctx context.Context, actor *models.Actor, userID string, roleIDs []string, assignedByUserID *string) error {
	return a.userRolesService.ReplaceUserRoles(ctx, actor, userID, roleIDs, assignedByUserID)
}

// User permissions

func (a *API) GetUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	return a.userPermissionsService.GetUserPermissions(ctx, actor, userID)
}

func (a *API) HasPermissions(ctx context.Context, actor *models.Actor, userID string, permissionNames []string) (bool, error) {
	return a.userPermissionsService.HasPermissions(ctx, actor, userID, permissionNames)
}
