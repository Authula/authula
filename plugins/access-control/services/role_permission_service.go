package services

import (
	"context"

	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type RolePermissionsService struct {
	rolesRepo           repositories.RolesRepository
	permissionsRepo     repositories.PermissionsRepository
	rolePermissionsRepo repositories.RolePermissionsRepository
}

func NewRolePermissionsService(rolesRepo repositories.RolesRepository, permissionsRepo repositories.PermissionsRepository, rolePermissionsRepo repositories.RolePermissionsRepository) *RolePermissionsService {
	return &RolePermissionsService{rolesRepo: rolesRepo, permissionsRepo: permissionsRepo, rolePermissionsRepo: rolePermissionsRepo}
}

func (s *RolePermissionsService) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	if roleID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}

	return s.rolePermissionsRepo.GetRolePermissions(ctx, roleID)
}

func (s *RolePermissionsService) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	if roleID == "" {
		return constants.ErrBadRequest
	}
	if permissionID == "" {
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
		return constants.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return constants.ErrNotFound
	}
	if permission.IsSystem {
		return constants.ErrBadRequest
	}

	return s.rolePermissionsRepo.AddRolePermission(ctx, roleID, permissionID, grantedByUserID)
}

func (s *RolePermissionsService) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	if roleID == "" {
		return constants.ErrUnprocessableEntity
	}
	if permissionID == "" {
		return constants.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return constants.ErrNotFound
	}
	if role.IsSystem {
		return constants.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return constants.ErrNotFound
	}
	if permission.IsSystem {
		return constants.ErrBadRequest
	}

	return s.rolePermissionsRepo.RemoveRolePermission(ctx, roleID, permissionID)
}

func (s *RolePermissionsService) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
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
		return constants.ErrBadRequest
	}

	normalized := make([]string, 0, len(permissionIDs))
	seen := make(map[string]struct{}, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		if permissionID == "" {
			continue
		}
		if _, ok := seen[permissionID]; ok {
			continue
		}
		seen[permissionID] = struct{}{}

		permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
		if err != nil {
			return err
		}
		if permission == nil {
			return constants.ErrNotFound
		}
		if permission.IsSystem {
			return constants.ErrBadRequest
		}

		normalized = append(normalized, permissionID)
	}

	return s.rolePermissionsRepo.ReplaceRolePermissions(ctx, roleID, normalized, grantedByUserID)
}
