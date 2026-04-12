package services

import (
	"context"
	"time"

	"github.com/Authula/authula/plugins/access-control/constants"
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
		return false, constants.ErrNotFound
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
		return false, constants.ErrForbidden
	}

	if role.Weight > highestWeight {
		return false, constants.ErrForbidden
	}

	return true, nil
}
