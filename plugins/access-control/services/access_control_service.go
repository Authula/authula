package services

import (
	"context"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
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
	role, err := s.rolesService.getRoleByNameInternal(ctx, roleName)
	if err != nil {
		return false, err
	}

	return role != nil && role.ID != "", nil
}

func (s *AccessControlService) ValidateRoleAssignment(ctx context.Context, roleName string, assignerUserID *string) (bool, error) {
	role, err := s.rolesService.getRoleByNameInternal(ctx, roleName)
	if err != nil {
		return false, err
	}
	if role == nil || role.ID == "" {
		return false, internalerrors.ErrNotFound
	}

	if assignerUserID == nil || *assignerUserID == "" {
		return false, nil
	}

	assignerRoles, err := s.userRolesService.getUserRolesInternal(ctx, *assignerUserID)
	if err != nil {
		return false, err
	}

	highestWeight, activeCount := determineHighestActiveRoleWeight(assignerRoles, time.Now().UTC())
	if activeCount == 0 {
		return false, internalerrors.ErrForbidden
	}

	if role.Weight > highestWeight {
		return false, internalerrors.ErrForbidden
	}

	return true, nil
}

func (s *AccessControlService) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	for _, key := range permissionKeys {
		permission, err := s.permissionsRepo.GetPermissionByKey(ctx, key)
		if err != nil {
			return err
		}
		if permission == nil {
			return internalerrors.ErrNotFound
		}
	}
	return nil
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
