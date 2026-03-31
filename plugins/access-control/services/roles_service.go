package services

import (
	"context"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type RolesService struct {
	rolesRepo           repositories.RolesRepository
	rolePermissionsRepo repositories.RolePermissionsRepository
	userRolesRepo       repositories.UserRolesRepository
}

func NewRolesService(rolesRepo repositories.RolesRepository, rolePermissionsRepo repositories.RolePermissionsRepository, userRolesRepo repositories.UserRolesRepository) *RolesService {
	return &RolesService{rolesRepo: rolesRepo, rolePermissionsRepo: rolePermissionsRepo, userRolesRepo: userRolesRepo}
}

func (s *RolesService) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	if req.Name == "" {
		return nil, constants.ErrBadRequest
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}

	role := &types.Role{
		ID:          util.GenerateUUID(),
		Name:        req.Name,
		Description: description,
		IsSystem:    req.IsSystem,
	}

	if err := s.rolesRepo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RolesService) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return s.rolesRepo.GetAllRoles(ctx)
}

func (s *RolesService) GetRoleByName(ctx context.Context, roleName string) (*types.Role, error) {
	if roleName == "" {
		return nil, constants.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByName(ctx, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}

	return role, nil
}

func (s *RolesService) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	if roleID == "" {
		return nil, constants.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}

	permissions, err := s.rolePermissionsRepo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	return &types.RoleDetails{Role: *role, Permissions: permissions}, nil
}

func (s *RolesService) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	if roleID == "" {
		return nil, constants.ErrBadRequest
	}

	if req.Name == nil && req.Description == nil {
		return nil, constants.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}
	if role.IsSystem {
		return nil, constants.ErrCannotUpdateSystemRole
	}

	var name *string
	if req.Name != nil {
		if *req.Name == "" {
			return nil, constants.ErrBadRequest
		}
		name = req.Name
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}

	updated, err := s.rolesRepo.UpdateRole(ctx, roleID, name, description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, constants.ErrNotFound
	}

	role, err = s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}

	return role, nil
}

func (s *RolesService) DeleteRole(ctx context.Context, roleID string) error {
	if roleID == "" {
		return constants.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return constants.ErrNotFound
	}
	if role.IsSystem {
		return constants.ErrCannotUpdateSystemRole
	}

	totalUsersByRole, err := s.userRolesRepo.CountUsersByRole(ctx, roleID)
	if err != nil {
		return err
	}
	if totalUsersByRole > 0 {
		return constants.ErrConflict
	}

	deleted, err := s.rolesRepo.DeleteRole(ctx, roleID)
	if err != nil {
		return err
	}
	if !deleted {
		return constants.ErrNotFound
	}

	return nil
}
