package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type UserPermissionsUseCase struct {
	service    *services.UserPermissionsService
	authorizer rootservices.Authorizer
}

func NewUserPermissionsUseCase(service *services.UserPermissionsService, authorizer rootservices.Authorizer) *UserPermissionsUseCase {
	return &UserPermissionsUseCase{service: service, authorizer: authorizer}
}

func (u *UserPermissionsUseCase) GetSelfUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	return u.service.GetSelfUserPermissions(ctx, actor, userID)
}

func (u *UserPermissionsUseCase) GetUserPermissions(ctx context.Context, actor *models.Actor, userID string) ([]types.UserPermissionInfo, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.UserPermissionsReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetUserPermissions(ctx, actor, userID)
}

func (u *UserPermissionsUseCase) HasPermissions(ctx context.Context, actor *models.Actor, userID string, permissionKeys []string) (bool, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.UserPermissionsCheckPermission); err != nil {
		return false, err
	}
	return u.service.HasPermissions(ctx, actor, userID, permissionKeys)
}
