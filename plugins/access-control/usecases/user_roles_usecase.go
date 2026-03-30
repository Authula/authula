package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserRolesUseCase struct {
	service *services.UserRolesService
}

func NewUserRolesUseCase(service *services.UserRolesService) *UserRolesUseCase {
	return &UserRolesUseCase{service: service}
}

func (u *UserRolesUseCase) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return u.service.GetUserRoles(ctx, userID)
}

func (u *UserRolesUseCase) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return u.service.GetUserWithRolesByID(ctx, userID)
}

func (u *UserRolesUseCase) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return u.service.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}

func (u *UserRolesUseCase) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return u.service.AssignRoleToUser(ctx, userID, req, assignedByUserID)
}

func (u *UserRolesUseCase) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return u.service.RemoveRoleFromUser(ctx, userID, roleID)
}
