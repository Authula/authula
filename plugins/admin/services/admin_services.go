package services

import "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"

type AdminServices struct {
	rolePermission *RolePermissionService
	userAccess     *UserAccessService
	impersonation  *ImpersonationService
	userState      *UserStateService
	sessionState   *SessionStateService
}

func NewAdminServices(repos *repositories.AdminRepositories) *AdminServices {
	return &AdminServices{
		rolePermission: NewRolePermissionService(repos.RolePermissionRepository()),
		userAccess:     NewUserAccessService(repos.UserAccessRepository()),
		impersonation:  NewImpersonationService(repos.ImpersonationRepository()),
		userState:      NewUserStateService(repos.UserStateRepository()),
		sessionState:   NewSessionStateService(repos.SessionStateRepository()),
	}
}

func (s *AdminServices) RolePermissionService() *RolePermissionService {
	return s.rolePermission
}

func (s *AdminServices) UserAccessService() *UserAccessService {
	return s.userAccess
}

func (s *AdminServices) ImpersonationService() *ImpersonationService {
	return s.impersonation
}

func (s *AdminServices) UserStateService() *UserStateService {
	return s.userState
}

func (s *AdminServices) SessionStateService() *SessionStateService {
	return s.sessionState
}
