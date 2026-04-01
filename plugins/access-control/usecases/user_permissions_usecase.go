package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserPermissionsUseCase struct {
	service *services.UserPermissionsService
}

func NewUserPermissionsUseCase(service *services.UserPermissionsService) *UserPermissionsUseCase {
	return &UserPermissionsUseCase{service: service}
}

func (u *UserPermissionsUseCase) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return u.service.GetUserPermissions(ctx, userID)
}

func (u *UserPermissionsUseCase) HasPermissions(ctx context.Context, userID string, permissionKeys []string) (bool, error) {
	return u.service.HasPermissions(ctx, userID, permissionKeys)
}
