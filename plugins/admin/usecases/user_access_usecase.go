package usecases

import (
	"context"
	"errors"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type userRolesUseCase struct {
	service *services.UserAccessService
}

func NewUserRolesUseCase(service *services.UserAccessService) UserRolesUseCase {
	return &userRolesUseCase{service: service}
}

func (u *userRolesUseCase) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	return u.service.GetUserRoles(ctx, userID)
}

func (u *userRolesUseCase) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	return u.service.GetUserEffectivePermissions(ctx, userID)
}

func (u *userRolesUseCase) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	permissions, err := u.GetUserEffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission.PermissionKey] = struct{}{}
	}

	for _, required := range requiredPermissions {
		required = strings.TrimSpace(required)
		if required == "" {
			continue
		}
		if _, ok := granted[required]; ok {
			return true, nil
		}
	}

	return false, nil
}

func (u *userRolesUseCase) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	return u.service.GetUserWithRolesByID(ctx, userID)
}

func (u *userRolesUseCase) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	return u.service.GetUserWithPermissionsByID(ctx, userID)
}

func (u *userRolesUseCase) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	withRoles, err := u.GetUserWithRolesByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if withRoles == nil {
		return nil, nil
	}

	withPermissions, err := u.GetUserWithPermissionsByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile := &types.UserAuthorizationProfile{
		User:  withRoles.User,
		Roles: withRoles.Roles,
	}
	if withPermissions != nil {
		profile.Permissions = withPermissions.Permissions
	}

	return profile, nil
}
