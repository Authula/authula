package services

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type UserAccessService struct {
	repo *repositories.UserAccessRepository
}

func NewUserAccessService(repo *repositories.UserAccessRepository) *UserAccessService {
	return &UserAccessService{repo: repo}
}

func (s *UserAccessService) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return s.repo.GetUserRoles(ctx, userID)
}

func (s *UserAccessService) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return s.repo.GetUserEffectivePermissions(ctx, userID)
}

func (s *UserAccessService) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return s.repo.GetUserWithRolesByID(ctx, userID)
}

func (s *UserAccessService) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return s.repo.GetUserWithPermissionsByID(ctx, userID)
}
