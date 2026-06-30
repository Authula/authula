package accesscontrol

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func NewAPI(useCases *usecases.UseCases) *API {
	return &API{useCases: useCases}
}

// Roles

func (a *API) GetAllRoles(ctx context.Context, actor *models.Actor) ([]types.Role, error) {
	return a.useCases.GetAllRoles(ctx, actor)
}

func (a *API) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	return a.useCases.GetRoleByName(ctx, actor, roleName)
}

func (a *API) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	return a.useCases.GetRoleByID(ctx, actor, roleID)
}

func (a *API) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	return a.useCases.CreateRole(ctx, actor, req)
}

func (a *API) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return a.useCases.UpdateRole(ctx, actor, roleID, req)
}

func (a *API) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	return a.useCases.DeleteRole(ctx, actor, roleID)
}

// Permissions

func (a *API) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	return a.useCases.CreatePermission(ctx, actor, req)
}

func (a *API) GetAllPermissions(ctx context.Context, actor *models.Actor) ([]types.Permission, error) {
	return a.useCases.GetAllPermissions(ctx, actor)
}

func (a *API) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	return a.useCases.GetPermissionByID(ctx, actor, permissionID)
}

func (a *API) GetRolePermissions(ctx context.Context, actor *models.Actor, roleID string) ([]types.UserPermissionInfo, error) {
	return a.useCases.GetRolePermissions(ctx, actor, roleID)
}

func (a *API) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return a.useCases.UpdatePermission(ctx, actor, permissionID, req)
}

func (a *API) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	return a.useCases.DeletePermission(ctx, actor, permissionID)
}

// Role permissions

func (a *API) AddPermissionToRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string, grantedByUserID *string) error {
	return a.useCases.AddPermissionToRole(ctx, actor, roleID, permissionID, grantedByUserID)
}

func (a *API) RemovePermissionFromRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string) error {
	return a.useCases.RemovePermissionFromRole(ctx, actor, roleID, permissionID)
}

func (a *API) ReplaceRolePermissions(ctx context.Context, actor *models.Actor, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return a.useCases.ReplaceRolePermissions(ctx, actor, roleID, permissionIDs, grantedByUserID)
}

// User roles

func (a *API) GetUserRoles(ctx context.Context, actor *models.Actor, userID string) ([]types.UserRoleInfo, error) {
	return a.useCases.GetUserRoles(ctx, actor, userID)
}

func (a *API) AssignRoleToUser(ctx context.Context, actor *models.Actor, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return a.useCases.AssignRoleToUser(ctx, actor, userID, req, assignedByUserID)
}

func (a *API) RemoveRoleFromUser(ctx context.Context, actor *models.Actor, userID string, roleID string) error {
	return a.useCases.RemoveRoleFromUser(ctx, actor, userID, roleID)
}

func (a *API) ReplaceUserRoles(ctx context.Context, actor *models.Actor, userID string, roleIDs []string, assignedByUserID *string) error {
	return a.useCases.ReplaceUserRoles(ctx, actor, userID, roleIDs, assignedByUserID)
}

// User permissions

func (a *API) GetUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	return a.useCases.GetUserPermissions(ctx, actor, userID)
}

func (a *API) HasPermissions(ctx context.Context, actor *models.Actor, userID string, permissionNames []string) (bool, error) {
	return a.useCases.HasPermissions(ctx, actor, userID, permissionNames)
}
