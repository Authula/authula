package usecases

import (
	"github.com/Authula/authula/plugins/admin/services"
	rootservices "github.com/Authula/authula/services"
)

type AdminUseCases struct {
	users         UsersUseCase
	accounts      AccountsUseCase
	state         StateUseCase
	impersonation ImpersonationUseCase
}

func NewAdminUseCases(
	usersService *services.UsersService,
	accountsService *services.AccountsService,
	stateService *services.StateService,
	impersonationService *services.ImpersonationService,
	authorizer rootservices.Authorizer,
) *AdminUseCases {
	return &AdminUseCases{
		users:         NewUsersUseCase(usersService, authorizer),
		accounts:      NewAccountsUseCase(accountsService, authorizer),
		state:         NewStateUseCase(stateService, authorizer),
		impersonation: NewImpersonationUseCase(stateService, impersonationService, authorizer),
	}
}

func (u *AdminUseCases) UsersUseCase() UsersUseCase {
	return u.users
}

func (u *AdminUseCases) StateUseCase() StateUseCase {
	return u.state
}

func (u *AdminUseCases) AccountsUseCase() AccountsUseCase {
	return u.accounts
}

func (u *AdminUseCases) ImpersonationUseCase() ImpersonationUseCase {
	return u.impersonation
}
