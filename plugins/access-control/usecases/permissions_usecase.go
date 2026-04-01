package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type PermissionsUseCase struct {
	service *services.PermissionsService
}

func NewPermissionsUseCase(service *services.PermissionsService) *PermissionsUseCase {
	return &PermissionsUseCase{service: service}
}

func (u *PermissionsUseCase) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.service.CreatePermission(ctx, req)
}

func (u *PermissionsUseCase) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return u.service.GetAllPermissions(ctx)
}

func (u *PermissionsUseCase) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	return u.service.GetPermissionByID(ctx, permissionID)
}

func (u *PermissionsUseCase) GetPermissionByKey(ctx context.Context, permissionKey string) (*types.Permission, error) {
	return u.service.GetPermissionByKey(ctx, permissionKey)
}

func (u *PermissionsUseCase) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.service.UpdatePermission(ctx, permissionID, req)
}

func (u *PermissionsUseCase) DeletePermission(ctx context.Context, permissionID string) error {
	return u.service.DeletePermission(ctx, permissionID)
}
