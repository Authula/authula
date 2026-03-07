package repositories

import "github.com/uptrace/bun"

type AdminRepositories struct {
	impersonation *BunImpersonationRepository
	userState     *BunUserStateRepository
	sessionState  *BunSessionStateRepository
}

func NewAdminRepositories(db bun.IDB) *AdminRepositories {
	return &AdminRepositories{
		impersonation: NewBunImpersonationRepository(db),
		userState:     NewBunUserStateRepository(db),
		sessionState:  NewBunSessionStateRepository(db),
	}
}

func (r *AdminRepositories) UserStateRepository() UserStateRepository {
	return r.userState
}

func (r *AdminRepositories) SessionStateRepository() SessionStateRepository {
	return r.sessionState
}

func (r *AdminRepositories) ImpersonationRepository() ImpersonationRepository {
	return r.impersonation
}
