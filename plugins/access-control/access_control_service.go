package accesscontrol

import "context"

type AccessControlService struct {
	api *API
}

func NewAccessControlService(api *API) *AccessControlService {
	return &AccessControlService{api: api}
}

func (s *AccessControlService) RoleExists(ctx context.Context, roleName string) (bool, error) {
	role, err := s.api.GetRoleByName(ctx, roleName)
	if err != nil {
		return false, err
	}

	return role != nil && role.ID != "", nil
}
