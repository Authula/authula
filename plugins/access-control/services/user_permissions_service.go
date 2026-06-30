package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type UserPermissionsService struct {
	repo       repositories.UserPermissionsRepository
	authorizer rootservices.Authorizer
}

func NewUserPermissionsService(repo repositories.UserPermissionsRepository, authorizer rootservices.Authorizer) *UserPermissionsService {
	return &UserPermissionsService{repo: repo, authorizer: authorizer}
}

func (s *UserPermissionsService) GetUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.UserPermissionsReadPermission); err != nil {
		return nil, err
	}

	if userID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	return s.repo.GetUserPermissions(ctx, userID)
}

func (s *UserPermissionsService) HasPermissions(ctx context.Context, actor *models.Actor, userID string, permissionKeys []string) (bool, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.UserPermissionsCheckPermission); err != nil {
		return false, err
	}

	if userID == "" {
		return false, internalerrors.ErrUnprocessableEntity
	}

	return s.repo.HasPermissions(ctx, userID, permissionKeys)
}
