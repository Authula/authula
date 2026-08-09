package admin

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
)

type API struct {
	usersService         *services.UsersService
	accountsService      *services.AccountsService
	stateService         *services.StateService
	impersonationService *services.ImpersonationService
	impersonationRepo    repositories.ImpersonationRepository
	userStateRepo        repositories.UserStateRepository
	sessionStateRepo     repositories.SessionStateRepository
}

func NewAPI(
	usersService *services.UsersService,
	accountsService *services.AccountsService,
	stateService *services.StateService,
	impersonationService *services.ImpersonationService,
	impersonationRepo repositories.ImpersonationRepository,
	userStateRepo repositories.UserStateRepository,
	sessionStateRepo repositories.SessionStateRepository,
) *API {
	return &API{
		usersService:         usersService,
		accountsService:      accountsService,
		stateService:         stateService,
		impersonationService: impersonationService,
		impersonationRepo:    impersonationRepo,
		userStateRepo:        userStateRepo,
		sessionStateRepo:     sessionStateRepo,
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
	return a.usersService.Create(ctx, actor, request)
}

func (a *API) GetAllUsers(ctx context.Context, actor *models.Actor, cursor *string, limit int) (*types.UsersPage, error) {
	return a.usersService.GetAll(ctx, actor, cursor, limit)
}

func (a *API) GetUserByID(ctx context.Context, actor *models.Actor, userID string) (*models.User, error) {
	return a.usersService.GetByID(ctx, actor, userID)
}

func (a *API) UpdateUser(ctx context.Context, actor *models.Actor, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return a.usersService.Update(ctx, actor, userID, request)
}

func (a *API) DeleteUser(ctx context.Context, actor *models.Actor, userID string) error {
	return a.usersService.Delete(ctx, actor, userID)
}

// Account management

func (a *API) CreateAccount(ctx context.Context, actor *models.Actor, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	return a.accountsService.Create(ctx, actor, userID, request)
}

func (a *API) GetAccountByID(ctx context.Context, actor *models.Actor, accountID string) (*models.Account, error) {
	return a.accountsService.GetByID(ctx, actor, accountID)
}

func (a *API) GetUserAccounts(ctx context.Context, actor *models.Actor, userID string) ([]models.Account, error) {
	return a.accountsService.GetByUserID(ctx, actor, userID)
}

func (a *API) UpdateAccount(ctx context.Context, actor *models.Actor, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	return a.accountsService.Update(ctx, actor, accountID, request)
}

func (a *API) DeleteAccount(ctx context.Context, actor *models.Actor, accountID string) error {
	return a.accountsService.Delete(ctx, actor, accountID)
}

// Impersonation

func (a *API) GetAllImpersonations(ctx context.Context, actor *models.Actor) ([]types.Impersonation, error) {
	return a.impersonationService.GetAllImpersonations(ctx, actor)
}

func (a *API) GetImpersonationByID(ctx context.Context, actor *models.Actor, impersonationID string) (*types.Impersonation, error) {
	return a.impersonationService.GetImpersonationByID(ctx, actor, impersonationID)
}

func (a *API) StartImpersonation(ctx context.Context, actor *models.Actor, actorSessionID *string, ipAddress *string, userAgent *string, req types.StartImpersonationRequest, impersonatorScopes []string, originalCookieValue string, originalCookieMaxAge int) (*types.StartImpersonationResult, error) {
	return a.impersonationService.StartImpersonation(ctx, actor, actorSessionID, ipAddress, userAgent, req, impersonatorScopes, originalCookieValue, originalCookieMaxAge)
}

func (a *API) StopImpersonation(ctx context.Context, actor *models.Actor, impersonatedSessionID string, originalCookieValue string, req types.StopImpersonationRequest) (*types.StopImpersonationResult, error) {
	return a.impersonationService.StopImpersonation(ctx, actor, impersonatedSessionID, originalCookieValue, req)
}

// User state

func (a *API) GetUserState(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return a.stateService.GetUserState(ctx, actor, userID)
}

func (a *API) UpsertUserState(ctx context.Context, actor *models.Actor, userID string, req types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.stateService.UpsertUserState(ctx, actor, userID, req, actorUserID)
}

func (a *API) CreateUserState(ctx context.Context, actor *models.Actor, userID string, req types.CreateUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.stateService.CreateUserState(ctx, actor, userID, req, actorUserID)
}

func (a *API) UpdateUserState(ctx context.Context, actor *models.Actor, userID string, req types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.stateService.UpdateUserState(ctx, actor, userID, req, actorUserID)
}

func (a *API) DeleteUserState(ctx context.Context, actor *models.Actor, userID string) error {
	return a.stateService.DeleteUserState(ctx, actor, userID)
}

func (a *API) GetBannedUserStates(ctx context.Context, actor *models.Actor) ([]types.AdminUserState, error) {
	return a.stateService.GetBannedUserStates(ctx, actor)
}

func (a *API) BanUser(ctx context.Context, actor *models.Actor, userID string, req types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	return a.stateService.BanUser(ctx, actor, userID, req, actorUserID)
}

func (a *API) UnbanUser(ctx context.Context, actor *models.Actor, userID string) (*types.AdminUserState, error) {
	return a.stateService.UnbanUser(ctx, actor, userID)
}

// Session state

func (a *API) GetSelfUserState(ctx context.Context, actor *models.Actor) (*types.AdminUserState, error) {
	return a.stateService.GetSelfUserState(ctx, actor)
}

func (a *API) GetSelfSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	return a.stateService.GetSelfSessionState(ctx, sessionID)
}

func (a *API) GetSessionState(ctx context.Context, actor *models.Actor, sessionID string) (*types.AdminSessionState, error) {
	return a.stateService.GetSessionState(ctx, actor, sessionID)
}

func (a *API) UpsertSessionState(ctx context.Context, actor *models.Actor, sessionID string, req types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.stateService.UpsertSessionState(ctx, actor, sessionID, req, actorUserID)
}

func (a *API) CreateSessionState(ctx context.Context, actor *models.Actor, sessionID string, req types.CreateSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.stateService.CreateSessionState(ctx, actor, sessionID, req, actorUserID)
}

func (a *API) UpdateSessionState(ctx context.Context, actor *models.Actor, sessionID string, req types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	return a.stateService.UpdateSessionState(ctx, actor, sessionID, req, actorUserID)
}

func (a *API) DeleteSessionState(ctx context.Context, actor *models.Actor, sessionID string) error {
	return a.stateService.DeleteSessionState(ctx, actor, sessionID)
}

func (a *API) RevokeSession(ctx context.Context, actor *models.Actor, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	return a.stateService.RevokeSession(ctx, actor, sessionID, reason, actorUserID)
}

func (a *API) GetUserAdminSessions(ctx context.Context, actor *models.Actor, userID string) ([]types.AdminUserSession, error) {
	return a.stateService.GetUserAdminSessions(ctx, actor, userID)
}

func (a *API) GetRevokedSessionStates(ctx context.Context, actor *models.Actor) ([]types.AdminSessionState, error) {
	return a.stateService.GetRevokedSessionStates(ctx, actor)
}
