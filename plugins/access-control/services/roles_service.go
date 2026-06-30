package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/models"
	rootservices "github.com/Authula/authula/services"
)

type RolesService struct {
	rolesRepo           repositories.RolesRepository
	rolePermissionsRepo repositories.RolePermissionsRepository
	userRolesRepo       repositories.UserRolesRepository
	authorizer          rootservices.Authorizer
}

func NewRolesService(rolesRepo repositories.RolesRepository, rolePermissionsRepo repositories.RolePermissionsRepository, userRolesRepo repositories.UserRolesRepository, authorizer rootservices.Authorizer) *RolesService {
	return &RolesService{rolesRepo: rolesRepo, rolePermissionsRepo: rolePermissionsRepo, userRolesRepo: userRolesRepo, authorizer: authorizer}
}

func (s *RolesService) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesCreatePermission); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, internalerrors.ErrBadRequest
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}
	weight := 10
	if req.Weight != nil {
		weight = *req.Weight
	}

	role := &types.Role{
		ID:          util.GenerateUUID(),
		Name:        req.Name,
		Description: description,
		Weight:      weight,
		IsSystem:    req.IsSystem,
	}

	if err := s.rolesRepo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RolesService) GetAllRoles(ctx context.Context, actor *models.Actor) ([]types.Role, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesListPermission); err != nil {
		return nil, err
	}

	return s.rolesRepo.GetAllRoles(ctx)
}

func (s *RolesService) getRoleByNameInternal(ctx context.Context, roleName string) (*types.Role, error) {
	if roleName == "" {
		return nil, internalerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByName(ctx, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, internalerrors.ErrNotFound
	}

	return role, nil
}

func (s *RolesService) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesReadPermission); err != nil {
		return nil, err
	}

	return s.getRoleByNameInternal(ctx, roleName)
}

func (s *RolesService) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesReadPermission); err != nil {
		return nil, err
	}

	if roleID == "" {
		return nil, internalerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, internalerrors.ErrNotFound
	}

	permissions, err := s.rolePermissionsRepo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	return &types.RoleDetails{Role: *role, Permissions: permissions}, nil
}

func (s *RolesService) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesUpdatePermission); err != nil {
		return nil, err
	}

	if roleID == "" {
		return nil, internalerrors.ErrBadRequest
	}

	if req.Name == nil && req.Description == nil && req.Weight == nil {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, internalerrors.ErrNotFound
	}
	if role.IsSystem {
		return nil, accesscontrolconstants.ErrCannotUpdateSystemRole
	}

	var name *string
	if req.Name != nil {
		if *req.Name == "" {
			return nil, internalerrors.ErrBadRequest
		}
		name = req.Name
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}

	var weight *int
	if req.Weight != nil {
		weight = req.Weight
	}

	updated, err := s.rolesRepo.UpdateRole(ctx, roleID, name, description, weight)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, internalerrors.ErrNotFound
	}

	role, err = s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, internalerrors.ErrNotFound
	}

	return role, nil
}

func (s *RolesService) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, accesscontrolconstants.RolesDeletePermission); err != nil {
		return err
	}

	if roleID == "" {
		return internalerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return internalerrors.ErrNotFound
	}
	if role.IsSystem {
		return accesscontrolconstants.ErrCannotUpdateSystemRole
	}

	totalUsersByRole, err := s.userRolesRepo.CountUsersByRole(ctx, roleID)
	if err != nil {
		return err
	}
	if totalUsersByRole > 0 {
		return internalerrors.ErrConflict
	}

	deleted, err := s.rolesRepo.DeleteRole(ctx, roleID)
	if err != nil {
		return err
	}
	if !deleted {
		return internalerrors.ErrNotFound
	}

	return nil
}
