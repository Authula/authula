package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
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

func (s *RolePermissionsService) GetRolePermissions(ctx context.Context, actor *models.Actor, roleID string) ([]types.UserPermissionInfo, error) {
	if roleID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, internalerrors.ErrNotFound
	}

	return s.rolePermissionsRepo.GetRolePermissions(ctx, roleID)
}

func (s *RolePermissionsService) AddPermissionToRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string, grantedByUserID *string) error {
	if roleID == "" {
		return internalerrors.ErrBadRequest
	}
	if permissionID == "" {
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
		return internalerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return internalerrors.ErrNotFound
	}
	if permission.IsSystem {
		return internalerrors.ErrBadRequest
	}

	return s.rolePermissionsRepo.AddRolePermission(ctx, roleID, permissionID, grantedByUserID)
}

func (s *RolePermissionsService) RemovePermissionFromRole(ctx context.Context, actor *models.Actor, roleID string, permissionID string) error {
	if roleID == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if permissionID == "" {
		return internalerrors.ErrUnprocessableEntity
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return internalerrors.ErrNotFound
	}
	if role.IsSystem {
		return internalerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return internalerrors.ErrNotFound
	}
	if permission.IsSystem {
		return internalerrors.ErrBadRequest
	}

	return s.rolePermissionsRepo.RemoveRolePermission(ctx, roleID, permissionID)
}

func (s *RolePermissionsService) ReplaceRolePermissions(ctx context.Context, actor *models.Actor, roleID string, permissionIDs []string, grantedByUserID *string) error {
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
		return internalerrors.ErrBadRequest
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
			return internalerrors.ErrNotFound
		}
		if permission.IsSystem {
			return internalerrors.ErrBadRequest
		}

		normalized = append(normalized, permissionID)
	}

	return s.rolePermissionsRepo.ReplaceRolePermissions(ctx, roleID, normalized, grantedByUserID)
}
