package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type PermissionsUseCase struct {
	service    *services.PermissionsService
	authorizer rootservices.Authorizer
}

func NewPermissionsUseCase(service *services.PermissionsService, authorizer rootservices.Authorizer) *PermissionsUseCase {
	return &PermissionsUseCase{service: service, authorizer: authorizer}
}

func (u *PermissionsUseCase) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsCreatePermission); err != nil {
		return nil, err
	}
	return u.service.CreatePermission(ctx, actor, req)
}

func (u *PermissionsUseCase) GetAllPermissions(ctx context.Context, actor *models.Actor) ([]types.Permission, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsListPermission); err != nil {
		return nil, err
	}
	return u.service.GetAllPermissions(ctx, actor)
}

func (u *PermissionsUseCase) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetPermissionByID(ctx, actor, permissionID)
}

func (u *PermissionsUseCase) GetPermissionByKey(ctx context.Context, actor *models.Actor, permissionKey string) (*types.Permission, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetPermissionByKey(ctx, actor, permissionKey)
}

func (u *PermissionsUseCase) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsUpdatePermission); err != nil {
		return nil, err
	}
	return u.service.UpdatePermission(ctx, actor, permissionID, req)
}

func (u *PermissionsUseCase) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsDeletePermission); err != nil {
		return err
	}
	return u.service.DeletePermission(ctx, actor, permissionID)
}
