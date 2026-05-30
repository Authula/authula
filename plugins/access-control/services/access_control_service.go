package services

import (
	"context"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
)

type AccessControlService struct {
	rolesService     *RolesService
	userRolesService *UserRolesService
}

func NewAccessControlService(rolesService *RolesService, userRolesService *UserRolesService) *AccessControlService {
	return &AccessControlService{rolesService: rolesService, userRolesService: userRolesService}
}

func (s *AccessControlService) RoleExists(ctx context.Context, roleName string) (bool, error) {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return false, err
	}

	return role != nil && role.ID != "", nil
}

func (s *AccessControlService) ValidateRoleAssignment(ctx context.Context, roleName string, assignerUserID *string) (bool, error) {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return false, err
	}
	if role == nil || role.ID == "" {
		return false, internalerrors.ErrNotFound
	}

	if assignerUserID == nil || *assignerUserID == "" {
		return false, nil
	}

	assignerRoles, err := s.userRolesService.GetUserRoles(ctx, *assignerUserID)
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
