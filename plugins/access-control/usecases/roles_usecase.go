package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type RolesUseCase struct {
	service *services.RolesService
}

func NewRolesUseCase(service *services.RolesService) *RolesUseCase {
	return &RolesUseCase{service: service}
}

func (u *RolesUseCase) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	return u.service.CreateRole(ctx, actor, req)
}

func (u *RolesUseCase) GetAllRoles(ctx context.Context, actor *models.Actor) ([]types.Role, error) {
	return u.service.GetAllRoles(ctx, actor)
}

func (u *RolesUseCase) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	return u.service.GetRoleByName(ctx, actor, roleName)
}

func (u *RolesUseCase) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	return u.service.GetRoleByID(ctx, actor, roleID)
}

func (u *RolesUseCase) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.service.UpdateRole(ctx, actor, roleID, req)
}

func (u *RolesUseCase) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	return u.service.DeleteRole(ctx, actor, roleID)
}
