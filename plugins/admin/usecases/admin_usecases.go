package usecases

import (
	"context"
	"time"

	corerepositories "github.com/Authula/authula/internal/repositories"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type AdminUseCases struct {
	users         UsersUseCase
	accounts      AccountsUseCase
	state         StateUseCase
	impersonation ImpersonationUseCase
}

func NewAdminUseCases(
	config types.AdminPluginConfig,
	userRepo corerepositories.UserRepository,
	accountRepo corerepositories.AccountRepository,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
	passwordService rootservices.PasswordService,
	userStateRepo repositories.UserStateRepository,
	sessionStateRepo repositories.SessionStateRepository,
	impersonationRepo repositories.ImpersonationRepository,
	sessionExpiresIn time.Duration,
	authorizer rootservices.Authorizer,
) *AdminUseCases {
	usersService := services.NewUsersService(userRepo)
	accountsService := services.NewAccountsService(accountRepo, userRepo, passwordService)
	impersonationService := services.NewImpersonationService(
		impersonationRepo,
		sessionStateRepo,
		sessionService,
		tokenService,
		sessionExpiresIn,
		config.ImpersonationMaxExpiresIn,
	)
	stateService := services.NewStateService(userStateRepo, sessionStateRepo, impersonationRepo)

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

func (u *AdminUseCases) CreateUser(ctx context.Context, actor *models.Actor, request types.CreateUserRequest) (*models.User, error) {
	return u.users.Create(ctx, actor, request)
}

func (u *AdminUseCases) GetAllUsers(ctx context.Context, actor *models.Actor, cursor *string, limit int) (*types.UsersPage, error) {
	return u.users.GetAll(ctx, actor, cursor, limit)
}

func (u *AdminUseCases) GetUserByID(ctx context.Context, actor *models.Actor, userID string) (*models.User, error) {
	return u.users.GetByID(ctx, actor, userID)
}

func (u *AdminUseCases) UpdateUser(ctx context.Context, actor *models.Actor, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return u.users.Update(ctx, actor, userID, request)
}

func (u *AdminUseCases) DeleteUser(ctx context.Context, actor *models.Actor, userID string) error {
	return u.users.Delete(ctx, actor, userID)
}

func (u *AdminUseCases) CreateAccount(ctx context.Context, actor *models.Actor, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	return u.accounts.Create(ctx, actor, userID, request)
}

func (u *AdminUseCases) GetAccountByID(ctx context.Context, actor *models.Actor, accountID string) (*models.Account, error) {
	return u.accounts.GetByID(ctx, actor, accountID)
}

func (u *AdminUseCases) GetUserAccounts(ctx context.Context, actor *models.Actor, userID string) ([]models.Account, error) {
	return u.accounts.GetByUserID(ctx, actor, userID)
}

func (u *AdminUseCases) UpdateAccount(ctx context.Context, actor *models.Actor, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	return u.accounts.Update(ctx, actor, accountID, request)
}

func (u *AdminUseCases) DeleteAccount(ctx context.Context, actor *models.Actor, accountID string) error {
	return u.accounts.Delete(ctx, actor, accountID)
}

func (u *AdminUseCases) GetAllImpersonations(ctx context.Context, actor *models.Actor) ([]types.Impersonation, error) {
	return u.impersonation.GetAllImpersonations(ctx, actor)
}

func (u *AdminUseCases) GetImpersonationByID(ctx context.Context, actor *models.Actor, impersonationID string) (*types.Impersonation, error) {
	return u.impersonation.GetImpersonationByID(ctx, actor, impersonationID)
}

func (u *AdminUseCases) StartImpersonation(ctx context.Context, actor *models.Actor, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest, impersonatorScopes []string, originalCookieValue string, originalCookieMaxAge int) (*types.StartImpersonationResult, error) {
	return u.impersonation.StartImpersonation(ctx, actor, actorUserID, actorSessionID, ipAddress, userAgent, req, impersonatorScopes, originalCookieValue, originalCookieMaxAge)
}

func (u *AdminUseCases) StopImpersonation(ctx context.Context, actor *models.Actor, impersonatedUserID string, impersonatedSessionID string, originalCookieValue string, request types.StopImpersonationRequest) (*types.StopImpersonationResult, error) {
	return u.impersonation.StopImpersonation(ctx, actor, impersonatedUserID, impersonatedSessionID, originalCookieValue, request)
}

func (u *AdminUseCases) GetUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return u.state.GetUserState(ctx, actor, userID)
}

func (u *AdminUseCases) UpsertUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.UpsertUserState(ctx, actor, userID, request, actorUserID)
}

func (u *AdminUseCases) CreateUserState(ctx context.Context, actor *models.Actor, userID string, request types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.CreateUserState(ctx, actor, userID, request, actorUserID)
}

func (u *AdminUseCases) UpdateUserState(ctx context.Context, actor *models.Actor, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.UpdateUserState(ctx, actor, userID, request, actorUserID)
}

func (u *AdminUseCases) DeleteUserState(ctx context.Context, actor *models.Actor, userID string) error {
	return u.state.DeleteUserState(ctx, actor, userID)
}

func (u *AdminUseCases) GetBannedUserStates(ctx context.Context, actor *models.Actor) ([]types.AdminUserState, error) {
	return u.state.GetBannedUserStates(ctx, actor)
}

func (u *AdminUseCases) BanUser(ctx context.Context, actor *models.Actor, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.BanUser(ctx, actor, userID, request, actorUserID)
}

func (u *AdminUseCases) UnbanUser(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return u.state.UnbanUser(ctx, actor, userID)
}

func (u *AdminUseCases) GetSelfUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return u.state.GetSelfUserState(ctx, actor, userID)
}

func (u *AdminUseCases) GetSelfSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return u.state.GetSelfSessionState(ctx, sessionID)
}

func (u *AdminUseCases) GetSessionState(ctx context.Context, actor *models.Actor, sessionID string) (*types.AdminSessionState, error) {
	return u.state.GetSessionState(ctx, actor, sessionID)
}

func (u *AdminUseCases) UpsertSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.UpsertSessionState(ctx, actor, sessionID, request, actorUserID)
}

func (u *AdminUseCases) CreateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.CreateSessionState(ctx, actor, sessionID, request, actorUserID)
}

func (u *AdminUseCases) UpdateSessionState(ctx context.Context, actor *models.Actor, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.UpdateSessionState(ctx, actor, sessionID, request, actorUserID)
}

func (u *AdminUseCases) DeleteSessionState(ctx context.Context, actor *models.Actor, sessionID string) error {
	return u.state.DeleteSessionState(ctx, actor, sessionID)
}

func (u *AdminUseCases) RevokeSession(ctx context.Context, actor *models.Actor, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.RevokeSession(ctx, actor, sessionID, reason, actorUserID)
}

func (u *AdminUseCases) GetUserAdminSessions(ctx context.Context, actor *models.Actor, userID string) ([]types.AdminUserSession, error) {
	return u.state.GetUserAdminSessions(ctx, actor, userID)
}

func (u *AdminUseCases) GetRevokedSessionStates(ctx context.Context, actor *models.Actor) ([]types.AdminSessionState, error) {
	return u.state.GetRevokedSessionStates(ctx, actor)
}
