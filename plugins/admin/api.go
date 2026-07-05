package admin

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
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

func (a *API) CreateUser(ctx context.Context, actor *models.Actor, request types.CreateUserRequest) (*models.User, error) {
	return a.useCases.CreateUser(ctx, actor, request)
}

func (a *API) GetAllUsers(ctx context.Context, actor *models.Actor, cursor *string, limit int) (*types.UsersPage, error) {
	return a.useCases.GetAllUsers(ctx, actor, cursor, limit)
}

func (a *API) GetUserByID(ctx context.Context, actor *models.Actor, userID string) (*models.User, error) {
	return a.useCases.GetUserByID(ctx, actor, userID)
}

func (a *API) UpdateUser(ctx context.Context, actor *models.Actor, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return a.useCases.UpdateUser(ctx, actor, userID, request)
}

func (a *API) DeleteUser(ctx context.Context, actor *models.Actor, userID string) error {
	return a.useCases.DeleteUser(ctx, actor, userID)
}

// Account management

func (a *API) CreateAccount(ctx context.Context, actor *models.Actor, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	return a.useCases.CreateAccount(ctx, actor, userID, request)
}

func (a *API) GetAccountByID(ctx context.Context, actor *models.Actor, accountID string) (*models.Account, error) {
	return a.useCases.GetAccountByID(ctx, actor, accountID)
}

func (a *API) GetUserAccounts(ctx context.Context, actor *models.Actor, userID string) ([]models.Account, error) {
	return a.useCases.GetUserAccounts(ctx, actor, userID)
}

func (a *API) UpdateAccount(ctx context.Context, actor *models.Actor, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	return a.useCases.UpdateAccount(ctx, actor, accountID, request)
}

func (a *API) DeleteAccount(ctx context.Context, actor *models.Actor, accountID string) error {
	return a.useCases.DeleteAccount(ctx, actor, accountID)
}

// Impersonation

func (a *API) GetAllImpersonations(ctx context.Context, actor *models.Actor) ([]types.Impersonation, error) {
	return a.useCases.GetAllImpersonations(ctx, actor)
}

func (a *API) GetImpersonationByID(ctx context.Context, actor *models.Actor, impersonationID string) (*types.Impersonation, error) {
	return a.useCases.GetImpersonationByID(ctx, actor, impersonationID)
}

func (a *API) StartImpersonation(ctx context.Context, actor *models.Actor, actorUserID string, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest, impersonatorScopes []string, originalCookieValue string, originalCookieMaxAge int) (*types.StartImpersonationResult, error) {
	return a.useCases.StartImpersonation(ctx, actor, actorUserID, actorSessionID, ipAddress, userAgent, req, impersonatorScopes, originalCookieValue, originalCookieMaxAge)
}

func (a *API) StopImpersonation(ctx context.Context, actor *models.Actor, impersonatedUserID string, impersonatedSessionID string, originalCookieValue string, req types.StopImpersonationRequest) (*types.StopImpersonationResult, error) {
	return a.useCases.StopImpersonation(ctx, actor, impersonatedUserID, impersonatedSessionID, originalCookieValue, req)
}

// User state

func (a *API) GetUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return a.useCases.GetUserState(ctx, actor, userID)
}

func (a *API) UpsertUserState(ctx context.Context, actor *models.Actor, userID string, req types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.UpsertUserState(ctx, actor, userID, req, actorUserID)
}

func (a *API) CreateUserState(ctx context.Context, actor *models.Actor, userID string, req types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.CreateUserState(ctx, actor, userID, req, actorUserID)
}

func (a *API) UpdateUserState(ctx context.Context, actor *models.Actor, userID string, req types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.UpdateUserState(ctx, actor, userID, req, actorUserID)
}

func (a *API) DeleteUserState(ctx context.Context, actor *models.Actor, userID string) error {
	return a.useCases.DeleteUserState(ctx, actor, userID)
}

func (a *API) GetBannedUserStates(ctx context.Context, actor *models.Actor) ([]types.AdminUserState, error) {
	return a.useCases.GetBannedUserStates(ctx, actor)
}

func (a *API) BanUser(ctx context.Context, actor *models.Actor, userID string, req types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.useCases.BanUser(ctx, actor, userID, req, actorUserID)
}

func (a *API) UnbanUser(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return a.useCases.UnbanUser(ctx, actor, userID)
}

// Session state

func (a *API) GetSelfUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return a.useCases.GetSelfUserState(ctx, actor, userID)
}

func (a *API) GetSelfSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return a.useCases.GetSelfSessionState(ctx, sessionID)
}

func (a *API) GetSessionState(ctx context.Context, actor *models.Actor, sessionID string) (*types.AdminSessionState, error) {
	return a.useCases.GetSessionState(ctx, actor, sessionID)
}

func (a *API) UpsertSessionState(ctx context.Context, actor *models.Actor, sessionID string, req types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.UpsertSessionState(ctx, actor, sessionID, req, actorUserID)
}

func (a *API) CreateSessionState(ctx context.Context, actor *models.Actor, sessionID string, req types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.CreateSessionState(ctx, actor, sessionID, req, actorUserID)
}

func (a *API) UpdateSessionState(ctx context.Context, actor *models.Actor, sessionID string, req types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.UpdateSessionState(ctx, actor, sessionID, req, actorUserID)
}

func (a *API) DeleteSessionState(ctx context.Context, actor *models.Actor, sessionID string) error {
	return a.useCases.DeleteSessionState(ctx, actor, sessionID)
}

func (a *API) RevokeSession(ctx context.Context, actor *models.Actor, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return a.useCases.RevokeSession(ctx, actor, sessionID, reason, actorUserID)
}

func (a *API) GetUserAdminSessions(ctx context.Context, actor *models.Actor, userID string) ([]types.AdminUserSession, error) {
	return a.useCases.GetUserAdminSessions(ctx, actor, userID)
}

func (a *API) GetRevokedSessionStates(ctx context.Context, actor *models.Actor) ([]types.AdminSessionState, error) {
	return a.useCases.GetRevokedSessionStates(ctx, actor)
}
