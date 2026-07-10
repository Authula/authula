package services

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
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

func (s *RolesService) CreateRole(ctx context.Context, actor *models.Actor, req types.CreateRoleRequest) (*types.Role, error) {
	if req.Name == "" {
		return nil, coreerrors.ErrBadRequest
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
	return s.rolesRepo.GetAllRoles(ctx)
}

func (s *RolesService) GetRoleByName(ctx context.Context, actor *models.Actor, roleName string) (*types.Role, error) {
	if roleName == "" {
		return nil, coreerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByName(ctx, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, coreerrors.ErrNotFound
	}

	return role, nil
}

func (s *RolesService) GetRoleByID(ctx context.Context, actor *models.Actor, roleID string) (*types.RoleDetails, error) {
	if roleID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, coreerrors.ErrNotFound
	}

	permissions, err := s.rolePermissionsRepo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	return &types.RoleDetails{Role: *role, Permissions: permissions}, nil
}

func (s *RolesService) UpdateRole(ctx context.Context, actor *models.Actor, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	if roleID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	if req.Name == nil && req.Description == nil && req.Weight == nil {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, coreerrors.ErrNotFound
	}
	if role.IsSystem {
		return nil, accesscontrolconstants.ErrCannotUpdateSystemRole
	}

	var name *string
	if req.Name != nil {
		if *req.Name == "" {
			return nil, coreerrors.ErrBadRequest
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
		return nil, coreerrors.ErrNotFound
	}

	role, err = s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, coreerrors.ErrNotFound
	}

	return role, nil
}

func (s *RolesService) DeleteRole(ctx context.Context, actor *models.Actor, roleID string) error {
	if roleID == "" {
		return coreerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return coreerrors.ErrNotFound
	}
	if role.IsSystem {
		return accesscontrolconstants.ErrCannotUpdateSystemRole
	}

	totalUsersByRole, err := s.userRolesRepo.CountUsersByRole(ctx, roleID)
	if err != nil {
		return err
	}
	if totalUsersByRole > 0 {
		return coreerrors.ErrConflict
	}

	deleted, err := s.rolesRepo.DeleteRole(ctx, roleID)
	if err != nil {
		return err
	}
	if !deleted {
		return coreerrors.ErrNotFound
	}

	return nil
}
