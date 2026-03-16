package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type RolePermissionUseCase struct {
	service *services.RolePermissionService
}

func NewRolePermissionUseCase(service *services.RolePermissionService) RolePermissionUseCase {
	return RolePermissionUseCase{service: service}
}

func (u RolePermissionUseCase) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return u.service.CreateRole(ctx, req)
}

func (u RolePermissionUseCase) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return u.service.GetAllRoles(ctx)
}

func (u RolePermissionUseCase) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return u.service.GetRoleByID(ctx, roleID)
}

func (u RolePermissionUseCase) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.service.UpdateRole(ctx, roleID, req)
}

func (u RolePermissionUseCase) DeleteRole(ctx context.Context, roleID string) error {
	return u.service.DeleteRole(ctx, roleID)
}

func (u RolePermissionUseCase) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.service.CreatePermission(ctx, req)
}

func (u RolePermissionUseCase) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return u.service.GetAllPermissions(ctx)
}

func (u RolePermissionUseCase) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.service.UpdatePermission(ctx, permissionID, req)
}

func (u RolePermissionUseCase) DeletePermission(ctx context.Context, permissionID string) error {
	return u.service.DeletePermission(ctx, permissionID)
}

func (u RolePermissionUseCase) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return u.service.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (u RolePermissionUseCase) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return u.service.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (u RolePermissionUseCase) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return u.service.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

func (u RolePermissionUseCase) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.service.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

func (u RolePermissionUseCase) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.service.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (u RolePermissionUseCase) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return u.service.RemoveRoleFromUser(ctx, userID, roleID)
}
