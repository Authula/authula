package services

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserPermissionsService struct {
	repo repositories.UserPermissionsRepository
}

func NewUserPermissionsService(repo repositories.UserPermissionsRepository) *UserPermissionsService {
	return &UserPermissionsService{repo: repo}
}

func (s *UserPermissionsService) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	if userID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	return s.repo.GetUserPermissions(ctx, userID)
}

func (s *UserPermissionsService) HasPermissions(ctx context.Context, userID string, permissionKeys []string) (bool, error) {
	if userID == "" {
		return false, constants.ErrUnprocessableEntity
	}

	return s.repo.HasPermissions(ctx, userID, permissionKeys)
}
