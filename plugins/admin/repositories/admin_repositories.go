package repositories

import "github.com/uptrace/bun"

type AdminRepositories struct {
	rolePermission *BunRolePermissionRepository
	userAccess     *BunUserAccessRepository
	impersonation  *BunImpersonationRepository
	userState      *BunUserStateRepository
	sessionState   *BunSessionStateRepository
}

func NewAdminRepositories(db bun.IDB) *AdminRepositories {
	return &AdminRepositories{
		rolePermission: NewBunRolePermissionRepository(db),
		userAccess:     NewBunUserAccessRepository(db),
		impersonation:  NewBunImpersonationRepository(db),
		userState:      NewBunUserStateRepository(db),
		sessionState:   NewBunSessionStateRepository(db),
	}
}

func (r *AdminRepositories) RolePermissionRepository() RolePermissionRepository {
	return r.rolePermission
}

func (r *AdminRepositories) UserAccessRepository() UserAccessRepository {
	return r.userAccess
}

func (r *AdminRepositories) ImpersonationRepository() ImpersonationRepository {
	return r.impersonation
}

func (r *AdminRepositories) UserStateRepository() UserStateRepository {
	return r.userState
}

func (r *AdminRepositories) SessionStateRepository() SessionStateRepository {
	return r.sessionState
}
