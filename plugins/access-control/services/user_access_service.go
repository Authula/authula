package services

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserAccessService struct {
	userRolesRepo  repositories.UserRolesRepository
	userAccessRepo repositories.UserAccessRepository
}

func NewUserAccessService(userRolesRepo repositories.UserRolesRepository, userAccessRepo repositories.UserAccessRepository) *UserAccessService {
	return &UserAccessService{userRolesRepo: userRolesRepo, userAccessRepo: userAccessRepo}
}

func (s *UserAccessService) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	if userID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	return s.userAccessRepo.GetUserWithPermissionsByID(ctx, userID)
}

func (s *UserAccessService) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	if userID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	withRoles, err := s.userRolesRepo.GetUserWithRolesByID(ctx, userID)
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
	if userID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	return s.userAccessRepo.GetUserEffectivePermissions(ctx, userID)
}

func (s *UserAccessService) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	if userID == "" {
		return false, constants.ErrUnprocessableEntity
	}

	permissions, err := s.GetUserEffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission.PermissionKey] = struct{}{}
	}

	for _, required := range requiredPermissions {
		if required == "" {
			continue
		}
		if _, ok := granted[required]; ok {
			return true, nil
		}
	}

	return false, nil
}
