package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserRolesUseCase struct {
	service *services.UserRolesService
}

func NewUserRolesUseCase(service *services.UserRolesService) *UserRolesUseCase {
	return &UserRolesUseCase{service: service}
}

func (u *UserRolesUseCase) GetUserRoles(ctx context.Context, actor *models.Actor, userID string) ([]types.UserRoleInfo, error) {
	return u.service.GetUserRoles(ctx, actor, userID)
}

func (u *UserRolesUseCase) ReplaceUserRoles(ctx context.Context, actor *models.Actor, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.service.ReplaceUserRoles(ctx, actor, userID, roleIDs, assignedByUserID)
}

func (u *UserRolesUseCase) AssignRoleToUser(ctx context.Context, actor *models.Actor, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.service.AssignRoleToUser(ctx, actor, userID, req, assignedByUserID)
}

func (u *UserRolesUseCase) RemoveRoleFromUser(ctx context.Context, actor *models.Actor, userID string, roleID string) error {
	return u.service.RemoveRoleFromUser(ctx, actor, userID, roleID)
}
