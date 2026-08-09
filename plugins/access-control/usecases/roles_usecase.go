package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type RolesUseCase struct {
	service    *services.RolesService
	authorizer rootservices.Authorizer
}

func NewRolesUseCase(service *services.RolesService, authorizer rootservices.Authorizer) *RolesUseCase {
	return &RolesUseCase{service: service, authorizer: authorizer}
}

func (u *RolesUseCase) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesCreatePermission); err != nil {
		return nil, err
	}
	return u.service.CreateRole(ctx, req)
}

func (u *RolesUseCase) GetAllRoles(ctx context.Context, actor *models.Actor) ([]types.Role, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesListPermission); err != nil {
		return nil, err
	}
	return u.service.GetAllRoles(ctx)
}

func (u *RolesUseCase) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetRoleByName(ctx, roleName)
}

func (u *RolesUseCase) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetRoleByID(ctx, roleID)
}

func (u *RolesUseCase) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesUpdatePermission); err != nil {
		return nil, err
	}
	return u.service.UpdateRole(ctx, roleID, req)
}

func (u *RolesUseCase) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesDeletePermission); err != nil {
		return err
	}
	return u.service.DeleteRole(ctx, roleID)
}
