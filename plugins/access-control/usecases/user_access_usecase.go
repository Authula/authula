package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserAccessUseCase struct {
	service *services.UserAccessService
}

func NewUserAccessUseCase(service *services.UserAccessService) *UserAccessUseCase {
	return &UserAccessUseCase{service: service}
}

func (u *UserAccessUseCase) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return u.service.GetUserWithPermissionsByID(ctx, userID)
}

func (u *UserAccessUseCase) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	return u.service.GetUserAuthorizationProfile(ctx, userID)
}

func (u *UserAccessUseCase) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return u.service.GetUserEffectivePermissions(ctx, userID)
}

func (u *UserAccessUseCase) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	return u.service.HasPermissions(ctx, userID, requiredPermissions)
}
