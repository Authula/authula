package services

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type AccessControlService struct {
	rolesService     *RolesService
	userRolesService *UserRolesService
	permissionsRepo  repositories.PermissionsRepository
}

func NewAccessControlService(rolesService *RolesService, userRolesService *UserRolesService, permissionsRepo repositories.PermissionsRepository) *AccessControlService {
	return &AccessControlService{rolesService: rolesService, userRolesService: userRolesService, permissionsRepo: permissionsRepo}
}

func (s *AccessControlService) RoleExists(ctx context.Context, roleName string) (bool, error) {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return false, err
	}

	return role != nil && role.ID != "", nil
}

func (s *AccessControlService) GetRolePermissionsByName(ctx context.Context, roleName string) ([]string, error) {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil || role.ID == "" {
		return nil, coreerrors.ErrNotFound
	}

	details, err := s.rolesService.GetRoleByID(ctx, role.ID)
	if err != nil {
		return nil, err
	}

	permissions := make([]string, 0, len(details.Permissions))
	for _, permission := range details.Permissions {
		permissions = append(permissions, permission.PermissionKey)
	}

	return permissions, nil
}

func (s *AccessControlService) GetRoleWeightByName(ctx context.Context, roleName string) (int, error) {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return 0, err
	}
	if role == nil || role.ID == "" {
		return 0, coreerrors.ErrNotFound
	}

	return role.Weight, nil
}

func (s *AccessControlService) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	for _, key := range permissionKeys {
		permission, err := s.permissionsRepo.GetPermissionByKey(ctx, key)
		if err != nil {
			return err
		}
		if permission == nil {
			return coreerrors.ErrNotFound
		}
	}
	return nil
}

func (s *AccessControlService) AssignRoleToUserIfMissing(ctx context.Context, userID string, roleName string, assignedByUserID *string) error {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return err
	}

	userRoles, err := s.userRolesService.GetUserRoles(ctx, userID)
	if err != nil {
		return err
	}

	for _, ur := range userRoles {
		if ur.RoleName == roleName {
			return nil
		}
	}

	return s.userRolesService.AssignRoleToUser(ctx, userID, types.AssignUserRoleRequest{RoleID: role.ID}, assignedByUserID)
}

func (s *AccessControlService) EnsurePermissions(ctx context.Context, permissions []rootservices.PermissionDefinition) error {
	for _, p := range permissions {
		existing, err := s.permissionsRepo.GetPermissionByKey(ctx, p.Key)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}

		desc := p.Description
		permission := &types.Permission{
			ID:          util.GenerateUUID(),
			Key:         p.Key,
			Description: &desc,
			IsSystem:    true,
		}
		if err := s.permissionsRepo.CreatePermission(ctx, permission); err != nil {
			return err
		}
	}
	return nil
}
