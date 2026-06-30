package services

import (
	"context"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/plugins/access-control/repositories"
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
