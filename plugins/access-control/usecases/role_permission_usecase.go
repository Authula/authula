package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type RolePermissionsUseCase struct {
	service *services.RolePermissionsService
}

func NewRolePermissionsUseCase(service *services.RolePermissionsService) *RolePermissionsUseCase {
	return &RolePermissionsUseCase{service: service}
}

func (u *RolePermissionsUseCase) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	return u.service.GetRolePermissions(ctx, roleID)
}

func (u *RolePermissionsUseCase) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return u.service.AddPermissionToRole(ctx, roleID, permissionID, grantedByUserID)
}

func (u *RolePermissionsUseCase) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return u.service.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (u *RolePermissionsUseCase) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return u.service.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}
