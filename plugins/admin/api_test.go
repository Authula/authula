package admin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
)

type apiTestFixture struct {
	api              *API
	userRepo         *internaltests.MockUserRepository
	accountRepo      *internaltests.MockAccountRepository
	userStateRepo    *admintests.MockUserStateRepository
	sessionStateRepo *admintests.MockSessionStateRepository
	impersonationRep *admintests.MockImpersonationRepository
	sessionSvc       *internaltests.MockSessionService
	tokenSvc         *internaltests.MockTokenService
	passwordSvc      *admintests.MockPasswordService
}

func newAPITestFixture() *apiTestFixture {
	f := &apiTestFixture{
		userRepo:         &internaltests.MockUserRepository{},
		accountRepo:      &internaltests.MockAccountRepository{},
		userStateRepo:    &admintests.MockUserStateRepository{},
		sessionStateRepo: &admintests.MockSessionStateRepository{},
		impersonationRep: &admintests.MockImpersonationRepository{},
		sessionSvc:       &internaltests.MockSessionService{},
		tokenSvc:         &internaltests.MockTokenService{},
		passwordSvc:      &admintests.MockPasswordService{},
	}

	usersService := adminservices.NewUsersService(f.userRepo)
	accountsService := adminservices.NewAccountsService(f.accountRepo, f.userRepo, f.passwordSvc)
	impersonationService := adminservices.NewImpersonationService(f.impersonationRep, f.sessionStateRepo, f.sessionSvc, f.tokenSvc, time.Minute, time.Minute)
	stateService := adminservices.NewStateService(f.userStateRepo, f.sessionStateRepo, f.impersonationRep)

	f.api = NewAPI(usersService, accountsService, stateService, impersonationService, f.impersonationRep, f.userStateRepo, f.sessionStateRepo)
	return f
}

func testAdminUseCases() *usecases.AdminUseCases {
	f := newAPITestFixture()
	return usecases.NewAdminUseCases(
		adminservices.NewUsersService(f.userRepo),
		adminservices.NewAccountsService(f.accountRepo, f.userRepo, f.passwordSvc),
		adminservices.NewStateService(f.userStateRepo, f.sessionStateRepo, f.impersonationRep),
		adminservices.NewImpersonationService(f.impersonationRep, f.sessionStateRepo, f.sessionSvc, f.tokenSvc, time.Minute, time.Minute),
		&internaltests.NoopAuthorizer{},
	)
}

func TestAPI_RepositoryAccessors(t *testing.T) {
	t.Parallel()

	f := newAPITestFixture()

	assert.Equal(t, f.impersonationRep, f.api.ImpersonationRepository())
	assert.Equal(t, f.userStateRepo, f.api.UserStateRepository())
	assert.Equal(t, f.sessionStateRepo, f.api.SessionStateRepository())
}

func TestAPI_UserManagement(t *testing.T) {
	t.Parallel()

	f := newAPITestFixture()
	ctx := context.Background()
	actor := internaltests.TestActor()

	f.userRepo.On("GetByEmail", mock.Anything, "alice@example.com").Return((*models.User)(nil), nil).Once()
	f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(&models.User{ID: "u1", Email: "alice@example.com"}, nil).Once()
	created, err := f.api.CreateUser(ctx, actor, types.CreateUserRequest{Name: "Alice", Email: "alice@example.com"})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "u1", created.ID)

	f.userRepo.On("GetAll", mock.Anything, (*string)(nil), 10).Return([]models.User{{ID: "u1"}}, (*string)(nil), nil).Once()
	page, err := f.api.GetAllUsers(ctx, actor, nil, 10)
	require.NoError(t, err)
	require.NotNil(t, page)
	assert.Len(t, page.Users, 1)

	f.userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	got, err := f.api.GetUserByID(ctx, actor, "u1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "u1", got.ID)

	f.userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1", Name: "Old"}, nil).Once()
	f.userRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(&models.User{ID: "u1", Name: "New"}, nil).Once()
	updated, err := f.api.UpdateUser(ctx, actor, "u1", types.UpdateUserRequest{Name: new("New")})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "New", updated.Name)

	f.userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	f.userRepo.On("Delete", mock.Anything, "u1").Return(nil).Once()
	require.NoError(t, f.api.DeleteUser(ctx, actor, "u1"))

	f.userRepo.AssertExpectations(t)
}

func TestAPI_AccountManagement(t *testing.T) {
	t.Parallel()

	f := newAPITestFixture()
	ctx := context.Background()
	actor := internaltests.TestActor()

	f.userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	f.accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
	f.accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()
	created, err := f.api.CreateAccount(ctx, actor, "u1", types.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "acc-1", created.ID)

	f.userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	f.accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return([]models.Account{{ID: "acc-1", UserID: "u1"}}, nil).Once()
	accounts, err := f.api.GetUserAccounts(ctx, actor, "u1")
	require.NoError(t, err)
	assert.Len(t, accounts, 1)

	f.accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
	got, err := f.api.GetAccountByID(ctx, actor, "acc-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "acc-1", got.ID)

	f.accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
	f.accountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Account")).Return(&models.Account{ID: "acc-1", Scope: new("openid")}, nil).Once()
	updated, err := f.api.UpdateAccount(ctx, actor, "acc-1", types.UpdateAccountRequest{Scope: new("openid")})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "openid", *updated.Scope)

	f.accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
	f.accountRepo.On("Delete", mock.Anything, "acc-1").Return(nil).Once()
	require.NoError(t, f.api.DeleteAccount(ctx, actor, "acc-1"))

	f.userRepo.AssertExpectations(t)
	f.accountRepo.AssertExpectations(t)
}

func TestAPI_UserStateManagement(t *testing.T) {
	t.Parallel()

	f := newAPITestFixture()
	ctx := context.Background()
	actor := internaltests.TestActor()
	actorID := actor.ID

	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	state, err := f.api.GetUserState(ctx, actor, "u1")
	require.NoError(t, err)
	require.NotNil(t, state)

	f.impersonationRep.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	f.userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	state, err = f.api.UpsertUserState(ctx, actor, "u1", types.UpsertUserStateRequest{}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.impersonationRep.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return((*types.AdminUserState)(nil), nil).Once()
	f.userStateRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	state, err = f.api.CreateUserState(ctx, actor, "u1", types.CreateUserStateRequest{}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.impersonationRep.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	f.userStateRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	state, err = f.api.UpdateUserState(ctx, actor, "u1", types.UpsertUserStateRequest{}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.userStateRepo.On("Delete", mock.Anything, "u1").Return(nil).Once()
	require.NoError(t, f.api.DeleteUserState(ctx, actor, "u1"))

	f.userStateRepo.On("GetBanned", mock.Anything).Return([]types.AdminUserState{{UserID: "u1", Banned: true}}, nil).Once()
	banned, err := f.api.GetBannedUserStates(ctx, actor)
	require.NoError(t, err)
	assert.Len(t, banned, 1)

	f.impersonationRep.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	f.userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1", Banned: true}, nil).Once()
	state, err = f.api.BanUser(ctx, actor, "u1", types.BanUserRequest{Reason: new("reason")}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.impersonationRep.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	f.userStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	f.userStateRepo.On("GetByUserID", mock.Anything, "u1").Return(&types.AdminUserState{UserID: "u1"}, nil).Once()
	state, err = f.api.UnbanUser(ctx, actor, "u1")
	require.NoError(t, err)
	require.NotNil(t, state)

	f.userStateRepo.On("GetByUserID", mock.Anything, actorID).Return(&types.AdminUserState{UserID: actorID}, nil).Once()
	self, err := f.api.GetSelfUserState(ctx, actor)
	require.NoError(t, err)
	require.NotNil(t, self)

	f.userStateRepo.AssertExpectations(t)
	f.impersonationRep.AssertExpectations(t)
}

func TestAPI_SessionStateManagement(t *testing.T) {
	t.Parallel()

	f := newAPITestFixture()
	ctx := context.Background()
	actor := internaltests.TestActor()
	actorID := actor.ID

	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	selfState, err := f.api.GetSelfSessionState(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, selfState)

	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	state, err := f.api.GetSessionState(ctx, actor, "s1")
	require.NoError(t, err)
	require.NotNil(t, state)

	f.sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
	f.sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	state, err = f.api.UpsertSessionState(ctx, actor, "s1", types.UpsertSessionStateRequest{}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return((*types.AdminSessionState)(nil), nil).Once()
	f.sessionStateRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	state, err = f.api.CreateSessionState(ctx, actor, "s1", types.CreateSessionStateRequest{}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	f.sessionStateRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	state, err = f.api.UpdateSessionState(ctx, actor, "s1", types.UpsertSessionStateRequest{}, &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.sessionStateRepo.On("Delete", mock.Anything, "s1").Return(nil).Once()
	require.NoError(t, f.api.DeleteSessionState(ctx, actor, "s1"))

	f.sessionStateRepo.On("SessionExists", mock.Anything, "s1").Return(true, nil).Once()
	f.sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "s1").Return(&types.AdminSessionState{SessionID: "s1"}, nil).Once()
	state, err = f.api.RevokeSession(ctx, actor, "s1", new("reason"), &actorID)
	require.NoError(t, err)
	require.NotNil(t, state)

	f.impersonationRep.On("UserExists", mock.Anything, "u1").Return(true, nil).Once()
	f.sessionStateRepo.On("GetByUserID", mock.Anything, "u1").Return([]types.AdminUserSession{{Session: models.Session{ID: "s1", UserID: "u1"}}}, nil).Once()
	sessions, err := f.api.GetUserAdminSessions(ctx, actor, "u1")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	f.sessionStateRepo.On("GetRevoked", mock.Anything).Return([]types.AdminSessionState{{SessionID: "s1"}}, nil).Once()
	revoked, err := f.api.GetRevokedSessionStates(ctx, actor)
	require.NoError(t, err)
	assert.Len(t, revoked, 1)

	f.sessionStateRepo.AssertExpectations(t)
	f.impersonationRep.AssertExpectations(t)
}

func TestAPI_ImpersonationManagement(t *testing.T) {
	t.Parallel()

	f := newAPITestFixture()
	ctx := context.Background()
	actor := internaltests.TestActor()

	f.impersonationRep.On("GetAllImpersonations", mock.Anything).Return([]types.Impersonation{{ID: "imp-1"}}, nil).Once()
	all, err := f.api.GetAllImpersonations(ctx, actor)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	f.impersonationRep.On("GetImpersonationByID", mock.Anything, "imp-1").Return(&types.Impersonation{ID: "imp-1"}, nil).Once()
	byID, err := f.api.GetImpersonationByID(ctx, actor, "imp-1")
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, "imp-1", byID.ID)

	impersonator := &models.Actor{ID: "actor-1", Type: models.ActorUser, Scopes: []string{"scope"}}
	f.impersonationRep.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
	f.impersonationRep.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
	f.tokenSvc.On("Generate").Return("raw-token", nil).Once()
	f.tokenSvc.On("Hash", "raw-token").Return("hashed-token").Once()
	f.sessionSvc.On("Create", mock.Anything, "target-1", "hashed-token", mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "imp-sess"}, nil).Once()
	f.impersonationRep.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
	f.sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()

	started, err := f.api.StartImpersonation(
		ctx,
		impersonator,
		nil,
		new("127.0.0.1"),
		new("agent"),
		types.StartImpersonationRequest{TargetUserID: "target-1", Reason: "reason"},
		[]string{"scope"},
		"",
		0,
	)
	require.NoError(t, err)
	require.NotNil(t, started)
	assert.Equal(t, "target-1", started.TargetUserID)
	require.NotNil(t, started.SessionID)
	assert.Equal(t, "imp-sess", *started.SessionID)

	stopActor := &models.Actor{
		ID:     "target-1",
		Type:   models.ActorUser,
		Claims: map[string]any{adminconstants.ImpersonatorID: "actor-1"},
	}
	f.sessionStateRepo.On("GetBySessionID", mock.Anything, "imp-sess").Return(&types.AdminSessionState{SessionID: "imp-sess", ImpersonatorUserID: new("actor-1")}, nil).Once()
	f.impersonationRep.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(&types.Impersonation{ID: "imp-1", ActorUserID: "actor-1"}, nil).Once()
	f.impersonationRep.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(nil).Once()

	stopped, err := f.api.StopImpersonation(ctx, stopActor, "imp-sess", "", types.StopImpersonationRequest{})
	require.NoError(t, err)
	require.NotNil(t, stopped)

	f.impersonationRep.AssertExpectations(t)
	f.sessionStateRepo.AssertExpectations(t)
	f.sessionSvc.AssertExpectations(t)
	f.tokenSvc.AssertExpectations(t)
}
