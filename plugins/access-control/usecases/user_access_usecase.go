package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type UserRolesUseCase struct {
	service *services.UserAccessService
}

func NewUserRolesUseCase(service *services.UserAccessService) UserRolesUseCase {
	return UserRolesUseCase{service: service}
}

func (u UserRolesUseCase) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return u.service.GetUserRoles(ctx, userID)
}

func (u UserRolesUseCase) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return u.service.GetUserWithRolesByID(ctx, userID)
}

func (u UserRolesUseCase) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return u.service.GetUserWithPermissionsByID(ctx, userID)
}

func (u UserRolesUseCase) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	return u.service.GetUserAuthorizationProfile(ctx, userID)
}

func (u UserRolesUseCase) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return u.service.GetUserEffectivePermissions(ctx, userID)
}

func (u UserRolesUseCase) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	return u.service.HasPermissions(ctx, userID, requiredPermissions)
}
