package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

type RolesUseCase struct {
	service *services.RolesService
}

func NewRolesUseCase(service *services.RolesService) *RolesUseCase {
	return &RolesUseCase{service: service}
}

func (u *RolesUseCase) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return u.service.CreateRole(ctx, req)
}

func (u *RolesUseCase) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return u.service.GetAllRoles(ctx)
}

func (u *RolesUseCase) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return u.service.GetRoleByID(ctx, roleID)
}

func (u *RolesUseCase) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return u.service.UpdateRole(ctx, roleID, req)
}

func (u *RolesUseCase) DeleteRole(ctx context.Context, roleID string) error {
	return u.service.DeleteRole(ctx, roleID)
}
