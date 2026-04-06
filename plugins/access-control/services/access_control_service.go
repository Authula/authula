package services

import "context"

type AccessControlService struct {
	rolesService *RolesService
}

func NewAccessControlService(rolesService *RolesService) *AccessControlService {
	return &AccessControlService{rolesService: rolesService}
}

func (s *AccessControlService) RoleExists(ctx context.Context, roleName string) (bool, error) {
	role, err := s.rolesService.GetRoleByName(ctx, roleName)
	if err != nil {
		return false, err
	}

	return role != nil && role.ID != "", nil
}
