package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type RolePermissionsUseCase struct {
	service    *services.RolePermissionsService
	authorizer rootservices.Authorizer
}

func NewRolePermissionsUseCase(service *services.RolePermissionsService, authorizer rootservices.Authorizer) *RolePermissionsUseCase {
	return &RolePermissionsUseCase{service: service, authorizer: authorizer}
}

func (u *RolePermissionsUseCase) GetRolePermissions(ctx context.Context, actor *models.Actor, roleID string) ([]types.UserPermissionInfo, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.RolePermissionsReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetRolePermissions(ctx, actor, roleID)
}

func (u *RolePermissionsUseCase) AddPermissionToRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string, grantedByUserID *string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.RolePermissionsAssignPermission); err != nil {
		return err
	}
	return u.service.AddPermissionToRole(ctx, actor, roleID, permissionID, grantedByUserID)
}

func (u *RolePermissionsUseCase) RemovePermissionFromRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.RolePermissionsRemovePermission); err != nil {
		return err
	}
	return u.service.RemovePermissionFromRole(ctx, actor, roleID, permissionID)
}

func (u *RolePermissionsUseCase) ReplaceRolePermissions(ctx context.Context, actor *models.Actor, roleID string, permissionIDs []string, grantedByUserID *string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.RolePermissionsAssignPermission); err != nil {
		return err
	}
	return u.service.ReplaceRolePermissions(ctx, actor, roleID, permissionIDs, grantedByUserID)
}
