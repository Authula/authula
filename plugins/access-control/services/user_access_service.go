package services

import (
	"context"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type UserAccessService struct {
	userAccessRepo repositories.UserAccessRepository
}

func NewUserAccessService(repo repositories.UserAccessRepository) *UserAccessService {
	return &UserAccessService{userAccessRepo: repo}
}

func (s *UserAccessService) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, constants.ErrBadRequest
	}
	return s.userAccessRepo.GetUserRoles(ctx, userID)
}

func (s *UserAccessService) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, constants.ErrBadRequest
	}
	return s.userAccessRepo.GetUserWithRolesByID(ctx, userID)
}

func (s *UserAccessService) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, constants.ErrBadRequest
	}
	return s.userAccessRepo.GetUserWithPermissionsByID(ctx, userID)
}

func (s *UserAccessService) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	withRoles, err := s.GetUserWithRolesByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if withRoles == nil {
		return nil, nil
	}

	withPermissions, err := s.GetUserWithPermissionsByID(ctx, userID)
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

func (s *UserAccessService) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, constants.ErrBadRequest
	}
	return s.userAccessRepo.GetUserEffectivePermissions(ctx, userID)
}

func (s *UserAccessService) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	permissions, err := s.GetUserEffectivePermissions(ctx, userID)
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
