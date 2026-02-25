package repositories

import "github.com/uptrace/bun"

type AdminRepositories struct {
	rolePermission *RolePermissionRepository
	userAccess     *UserAccessRepository
	impersonation  *ImpersonationRepository
	userState      *UserStateRepository
	sessionState   *SessionStateRepository
}

func NewAdminRepositories(db bun.IDB) *AdminRepositories {
	return &AdminRepositories{
		rolePermission: NewRolePermissionRepository(db),
		userAccess:     NewUserAccessRepository(db),
		impersonation:  NewImpersonationRepository(db),
		userState:      NewUserStateRepository(db),
		sessionState:   NewSessionStateRepository(db),
	}
}

func (r *AdminRepositories) RolePermissionRepository() *RolePermissionRepository {
	return r.rolePermission
}

func (r *AdminRepositories) UserAccessRepository() *UserAccessRepository {
	return r.userAccess
}

func (r *AdminRepositories) ImpersonationRepository() *ImpersonationRepository {
	return r.impersonation
}

func (r *AdminRepositories) UserStateRepository() *UserStateRepository {
	return r.userState
}

func (r *AdminRepositories) SessionStateRepository() *SessionStateRepository {
	return r.sessionState
}
