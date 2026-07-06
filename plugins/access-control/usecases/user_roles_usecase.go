package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type UserRolesUseCase struct {
	service    *services.UserRolesService
	authorizer rootservices.Authorizer
}

func NewUserRolesUseCase(service *services.UserRolesService, authorizer rootservices.Authorizer) *UserRolesUseCase {
	return &UserRolesUseCase{service: service, authorizer: authorizer}
}

func (u *UserRolesUseCase) GetUserRoles(ctx context.Context, actor *models.Actor, userID string) ([]types.UserRoleInfo, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetUserRoles(ctx, actor, userID)
}

func (u *UserRolesUseCase) ReplaceUserRoles(ctx context.Context, actor *models.Actor, userID string, roleIDs []string, assignedByUserID *string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesAssignPermission); err != nil {
		return err
	}
	return u.service.ReplaceUserRoles(ctx, actor, userID, roleIDs, assignedByUserID)
}

func (u *UserRolesUseCase) AssignRoleToUser(ctx context.Context, actor *models.Actor, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesAssignPermission); err != nil {
		return err
	}
	return u.service.AssignRoleToUser(ctx, actor, userID, req, assignedByUserID)
}

func (u *UserRolesUseCase) RemoveRoleFromUser(ctx context.Context, actor *models.Actor, userID string, roleID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesRemovePermission); err != nil {
		return err
	}
	return u.service.RemoveRoleFromUser(ctx, actor, userID, roleID)
}
