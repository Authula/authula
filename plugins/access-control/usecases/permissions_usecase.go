package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type PermissionsUseCase struct {
	service *services.PermissionsService
}

func NewPermissionsUseCase(service *services.PermissionsService) *PermissionsUseCase {
	return &PermissionsUseCase{service: service}
}

func (u *PermissionsUseCase) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	return u.service.CreatePermission(ctx, actor, req)
}

func (u *PermissionsUseCase) GetAllPermissions(ctx context.Context, actor *models.Actor) ([]types.Permission, error) {
	return u.service.GetAllPermissions(ctx, actor)
}

func (u *PermissionsUseCase) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	return u.service.GetPermissionByID(ctx, actor, permissionID)
}

func (u *PermissionsUseCase) GetPermissionByKey(ctx context.Context, actor *models.Actor, permissionKey string) (*types.Permission, error) {
	return u.service.GetPermissionByKey(ctx, actor, permissionKey)
}

func (u *PermissionsUseCase) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return u.service.UpdatePermission(ctx, actor, permissionID, req)
}

func (u *PermissionsUseCase) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	return u.service.DeletePermission(ctx, actor, permissionID)
}
