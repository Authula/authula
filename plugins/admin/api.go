package admin

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type API struct {
	useCases          *usecases.AdminUseCases
	impersonationRepo repositories.ImpersonationRepository
	userStateRepo     repositories.UserStateRepository
	sessionStateRepo  repositories.SessionStateRepository
}

func NewAPI(
	useCases *usecases.AdminUseCases,
	impersonationRepo repositories.ImpersonationRepository,
	userStateRepo repositories.UserStateRepository,
	sessionStateRepo repositories.SessionStateRepository,
) *API {
	return &API{
		useCases:          useCases,
		impersonationRepo: impersonationRepo,
		userStateRepo:     userStateRepo,
		sessionStateRepo:  sessionStateRepo,
	}
}

func (a *API) ImpersonationRepository() repositories.ImpersonationRepository {
	return a.impersonationRepo
}

func (a *API) UserStateRepository() repositories.UserStateRepository {
	return a.userStateRepo
}

func (a *API) SessionStateRepository() repositories.SessionStateRepository {
	return a.sessionStateRepo
}

// User management

func (a *API) CreateUser(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	return a.useCases.CreateUser(ctx, request)
}

func (a *API) GetAllUsers(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	return a.useCases.GetAllUsers(ctx, cursor, limit)
}

func (a *API) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return a.useCases.GetUserByID(ctx, userID)
}

func (a *API) UpdateUser(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return a.useCases.UpdateUser(ctx, userID, request)
}

func (a *API) DeleteUser(ctx context.Context, userID string) error {
	return a.useCases.DeleteUser(ctx, userID)
}

// Account management

func (a *API) CreateAccount(ctx context.Context, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	return a.useCases.CreateAccount(ctx, userID, request)
}

func (a *API) GetAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	return a.useCases.GetAccountByID(ctx, accountID)
}

func (a *API) GetUserAccounts(ctx context.Context, userID string) ([]models.Account, error) {
	return a.useCases.GetUserAccounts(ctx, userID)
}

func (a *API) UpdateAccount(ctx context.Context, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	return a.useCases.UpdateAccount(ctx, accountID, request)
}

func (a *API) DeleteAccount(ctx context.Context, accountID string) error {
	return a.useCases.DeleteAccount(ctx, accountID)
}

// Impersonation

func (a *API) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return a.useCases.GetAllImpersonations(ctx)
}

func (a *API) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	return a.useCases.GetImpersonationByID(ctx, impersonationID)
}

func (a *API) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	return a.useCases.StartImpersonation(ctx, actorUserID, actorSessionID, ipAddress, userAgent, req)
}

func (a *API) StopImpersonation(ctx context.Context, impersonatedUserID string, impersonatedSessionID string, req types.StopImpersonationRequest) error {
	return a.useCases.StopImpersonation(ctx, impersonatedUserID, impersonatedSessionID, req)
}

// User state

func (a *API) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return a.useCases.GetUserState(ctx, userID)
}

func (a *API) UpsertUserState(ctx context.Context, userID string, req types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.UpsertUserState(ctx, userID, req, actorUserID)
}

func (a *API) DeleteUserState(ctx context.Context, userID string) error {
	return a.useCases.DeleteUserState(ctx, userID)
}

func (a *API) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	return a.useCases.GetBannedUserStates(ctx)
}

func (a *API) BanUser(ctx context.Context, userID string, req types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.BanUser(ctx, userID, req, actorUserID)
}

func (a *API) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	return a.useCases.UnbanUser(ctx, userID)
}

// Session state

func (a *API) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return a.useCases.GetSessionState(ctx, sessionID)
}

func (a *API) UpsertSessionState(ctx context.Context, sessionID string, req types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.UpsertSessionState(ctx, sessionID, req, actorUserID)
}

func (a *API) DeleteSessionState(ctx context.Context, sessionID string) error {
	return a.useCases.DeleteSessionState(ctx, sessionID)
}

func (a *API) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.RevokeSession(ctx, sessionID, reason, actorUserID)
}

func (a *API) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	return a.useCases.GetUserAdminSessions(ctx, userID)
}

func (a *API) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	return a.useCases.GetRevokedSessionStates(ctx)
}
