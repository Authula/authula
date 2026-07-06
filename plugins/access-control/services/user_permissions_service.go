package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserPermissionsService struct {
	repo repositories.UserPermissionsRepository
}

func NewUserPermissionsService(repo repositories.UserPermissionsRepository) *UserPermissionsService {
	return &UserPermissionsService{repo: repo}
}

func (s *UserPermissionsService) GetSelfUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	if actor.ID != userID {
		return nil, internalerrors.ErrForbidden
	}

	if userID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	return s.repo.GetUserPermissions(ctx, userID)
}

func (s *UserPermissionsService) GetUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	if userID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	return s.repo.GetUserPermissions(ctx, userID)
}

func (s *UserPermissionsService) HasPermissions(ctx context.Context, actor *models.Actor, userID string, permissionKeys []string) (bool, error) {
	if userID == "" {
		return false, internalerrors.ErrUnprocessableEntity
	}

	return s.repo.HasPermissions(ctx, userID, permissionKeys)
}
