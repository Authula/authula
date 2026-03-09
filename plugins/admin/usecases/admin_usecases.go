package usecases

import (
	"context"
	"time"

	corerepositories "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
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
		users:         NewUsersUseCase(usersService),
		accounts:      NewAccountsUseCase(accountsService),
		state:         NewStateUseCase(stateService),
		impersonation: NewImpersonationUseCase(stateService, impersonationService),
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

func (u *AdminUseCases) CreateUser(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	return u.users.Create(ctx, request)
}

func (u *AdminUseCases) GetAllUsers(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	return u.users.GetAll(ctx, cursor, limit)
}

func (u *AdminUseCases) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return u.users.GetByID(ctx, userID)
}

func (u *AdminUseCases) UpdateUser(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return u.users.Update(ctx, userID, request)
}

func (u *AdminUseCases) DeleteUser(ctx context.Context, userID string) error {
	return u.users.Delete(ctx, userID)
}

func (u *AdminUseCases) CreateAccount(ctx context.Context, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	return u.accounts.Create(ctx, userID, request)
}

func (u *AdminUseCases) GetAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	return u.accounts.GetByID(ctx, accountID)
}

func (u *AdminUseCases) GetUserAccounts(ctx context.Context, userID string) ([]models.Account, error) {
	return u.accounts.GetByUserID(ctx, userID)
}

func (u *AdminUseCases) UpdateAccount(ctx context.Context, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	return u.accounts.Update(ctx, accountID, request)
}

func (u *AdminUseCases) DeleteAccount(ctx context.Context, accountID string) error {
	return u.accounts.Delete(ctx, accountID)
}

func (u *AdminUseCases) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return u.impersonation.GetAllImpersonations(ctx)
}

func (u *AdminUseCases) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return u.impersonation.GetImpersonationByID(ctx, impersonationID)
}

func (u *AdminUseCases) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return u.impersonation.StartImpersonation(ctx, actorUserID, actorSessionID, ipAddress, userAgent, req)
}

func (u *AdminUseCases) StopImpersonation(ctx context.Context, impersonatedUserID string, impersonatedSessionID string, request types.StopImpersonationRequest) error {
	return u.impersonation.StopImpersonation(ctx, impersonatedUserID, impersonatedSessionID, request)
}

func (u *AdminUseCases) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return u.state.GetUserState(ctx, userID)
}

func (u *AdminUseCases) UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.UpsertUserState(ctx, userID, request, actorUserID)
}

func (u *AdminUseCases) DeleteUserState(ctx context.Context, userID string) error {
	return u.state.DeleteUserState(ctx, userID)
}

func (u *AdminUseCases) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return u.state.GetBannedUserStates(ctx)
}

func (u *AdminUseCases) BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return u.state.BanUser(ctx, userID, request, actorUserID)
}

func (u *AdminUseCases) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return u.state.UnbanUser(ctx, userID)
}

func (u *AdminUseCases) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return u.state.GetSessionState(ctx, sessionID)
}

func (u *AdminUseCases) UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.UpsertSessionState(ctx, sessionID, request, actorUserID)
}

func (u *AdminUseCases) DeleteSessionState(ctx context.Context, sessionID string) error {
	return u.state.DeleteSessionState(ctx, sessionID)
}

func (u *AdminUseCases) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return u.state.RevokeSession(ctx, sessionID, reason, actorUserID)
}

func (u *AdminUseCases) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	return u.state.GetUserAdminSessions(ctx, userID)
}

func (u *AdminUseCases) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return u.state.GetRevokedSessionStates(ctx)
}
