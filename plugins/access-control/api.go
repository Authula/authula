package accesscontrol

import (
	"context"

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

func (a *API) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return a.useCases.GetAllRoles(ctx)
}

func (a *API) GetRoleByName(ctx context.Context, roleName string) (*types.Role, error) {
	return a.useCases.GetRoleByName(ctx, roleName)
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

// Permissions

func (a *API) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return a.useCases.CreatePermission(ctx, req)
}

func (a *API) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return a.useCases.GetAllPermissions(ctx)
}

func (a *API) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	return a.useCases.GetPermissionByID(ctx, permissionID)
}

func (a *API) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	return a.useCases.GetRolePermissions(ctx, roleID)
}

func (a *API) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return a.useCases.UpdatePermission(ctx, permissionID, req)
}

func (a *API) DeletePermission(ctx context.Context, permissionID string) error {
	return a.useCases.DeletePermission(ctx, permissionID)
}

// Role permissions

func (a *API) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return a.useCases.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (a *API) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return a.useCases.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (a *API) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return a.useCases.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

// User roles

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

// User permissions

func (a *API) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return a.useCases.GetUserPermissions(ctx, userID)
}

func (a *API) HasPermissions(ctx context.Context, userID string, permissionNames []string) (bool, error) {
	return a.useCases.HasPermissions(ctx, userID, permissionNames)
}
