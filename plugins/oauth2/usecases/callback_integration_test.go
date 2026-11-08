package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	corerepos "github.com/Authula/authula/core/repositories"
	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type callbackExecutor interface {
	Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error)
}

type callbackIntegrationFixture struct {
	userService    rootservices.UserService
	accountService rootservices.AccountService
	sessionService rootservices.SessionService
	tokenService   rootservices.TokenService
	flowStateRepo  repositories.FlowStateRepository
	accountRepo    corerepos.AccountRepository
	tokenRepo      corerepos.TokenRepository
	sessionID      string
}

func newCallbackIntegrationFixture(t *testing.T) *callbackIntegrationFixture {
	t.Helper()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)

	for _, model := range []any{
		(*models.User)(nil),
		(*models.Account)(nil),
		(*models.Session)(nil),
		(*types.OAuthFlowState)(nil),
	} {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		require.NoError(t, err)
	}

	tokenRepo, err := corerepos.NewCryptoTokenRepository(internaltests.MustDecodeKey(internaltests.EncryptionKeyOne), nil)
	require.NoError(t, err)
	accountRepo := corerepos.NewBunAccountRepository(db)
	userRepo := corerepos.NewBunUserRepository(db)
	sessionRepo := corerepos.NewBunSessionRepository(db)

	return &callbackIntegrationFixture{
		userService:    coreservices.NewUserService(userRepo, nil, &internaltests.MockLogger{}),
		accountService: coreservices.NewAccountService(nil, accountRepo, tokenRepo, nil, &internaltests.MockLogger{}),
		sessionService: coreservices.NewSessionService(sessionRepo, nil, nil, &internaltests.MockLogger{}),
		tokenService:   coreservices.NewTokenService(tokenRepo),
		flowStateRepo:  repositories.NewBunFlowStateRepository(db),
		accountRepo:    accountRepo,
		tokenRepo:      tokenRepo,
	}
}

func (f *callbackIntegrationFixture) seedFlowState(t *testing.T, state string, binding string, purpose string, userID *string) {
	t.Helper()

	_, err := f.flowStateRepo.Create(context.Background(), &types.OAuthFlowState{
		ID:        util.GenerateUUID(),
		StateHash: util.HashWithSalt(state, binding),
		Provider:  "google",
		Purpose:   purpose,
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
	})
	require.NoError(t, err)
}

func (f *callbackIntegrationFixture) newUseCase(provider *oauth2tests.MockProvider) callbackExecutor {
	return usecases.NewCallbackUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}},
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		f.flowStateRepo,
		f.userService,
		f.accountService,
		f.sessionService,
		f.tokenService,
	)
}

func TestCallbackUseCase_SQLiteIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*testing.T, *callbackIntegrationFixture) (*oauth2tests.MockProvider, *types.CallbackRequest)
		verify func(*testing.T, *callbackIntegrationFixture, *types.CallbackResult)
	}{
		{
			name: "stores encrypted tokens for the correct owner",
			setup: func(t *testing.T, f *callbackIntegrationFixture) (*oauth2tests.MockProvider, *types.CallbackRequest) {
				t.Helper()

				provider := &oauth2tests.MockProvider{
					ProviderName: "google",
					Token: &oauth2.Token{
						AccessToken:  "plain-access-token",
						RefreshToken: "plain-refresh-token",
						Expiry:       time.Now().Add(time.Hour),
					},
					UserInfo: &types.UserInfo{
						ProviderAccountID: "google-1",
						Email:             "new@example.com",
						EmailVerified:     true,
						Name:              "New User",
					},
				}

				f.seedFlowState(t, "state-1", "binding-1", types.OAuthFlowPurposeLogin, nil)

				return provider, &types.CallbackRequest{
					ProviderID: "google",
					Code:       "code-1",
					State:      "state-1",
					Binding:    "binding-1",
				}
			},
			verify: func(t *testing.T, f *callbackIntegrationFixture, result *types.CallbackResult) {
				t.Helper()

				ctx := context.Background()
				require.NotNil(t, result.User)

				stored, err := f.accountRepo.GetByProviderAndAccountID(ctx, "google", "google-1")
				require.NoError(t, err)
				require.NotNil(t, stored)
				assert.Equal(t, result.User.ID, stored.UserID)
				require.NotNil(t, stored.AccessToken)
				require.NotNil(t, stored.RefreshToken)
				assert.NotEqual(t, "plain-access-token", *stored.AccessToken)
				assert.NotEqual(t, "plain-refresh-token", *stored.RefreshToken)

				decryptedAccess, err := f.tokenRepo.Decrypt(*stored.AccessToken)
				require.NoError(t, err)
				assert.Equal(t, "plain-access-token", decryptedAccess)

				decryptedRefresh, err := f.tokenRepo.Decrypt(*stored.RefreshToken)
				require.NoError(t, err)
				assert.Equal(t, "plain-refresh-token", decryptedRefresh)
			},
		},
		{
			name: "links a second provider while signed in",
			setup: func(t *testing.T, f *callbackIntegrationFixture) (*oauth2tests.MockProvider, *types.CallbackRequest) {
				t.Helper()

				ctx := context.Background()
				user, err := f.userService.Create(ctx, "Existing User", "existing@example.com", true, nil, nil)
				require.NoError(t, err)
				_, err = f.accountService.Create(ctx, user.ID, "existing@example.com", models.AuthProviderEmail.String(), new("hashed-password"))
				require.NoError(t, err)

				session, err := f.sessionService.Create(ctx, user.ID, f.tokenService.Hash("session-token"), nil, nil, time.Hour)
				require.NoError(t, err)
				f.sessionID = session.ID

				provider := &oauth2tests.MockProvider{
					ProviderName: "google",
					Token: &oauth2.Token{
						AccessToken:  "linked-access-token",
						RefreshToken: "linked-refresh-token",
						Expiry:       time.Now().Add(time.Hour),
					},
					UserInfo: &types.UserInfo{
						ProviderAccountID: "google-linked",
						Email:             "existing@example.com",
						EmailVerified:     true,
						Name:              "Existing User",
					},
				}

				f.seedFlowState(t, "state-link", "binding-link", types.OAuthFlowPurposeLink, &user.ID)

				return provider, &types.CallbackRequest{
					ProviderID: "google",
					Code:       "code-link",
					State:      "state-link",
					Binding:    "binding-link",
					ActorID:    user.ID,
					SessionID:  session.ID,
				}
			},
			verify: func(t *testing.T, f *callbackIntegrationFixture, result *types.CallbackResult) {
				t.Helper()

				ctx := context.Background()
				require.NotNil(t, result.Session)
				require.NotNil(t, result.User)
				assert.True(t, result.ReusedSession)
				assert.Equal(t, f.sessionID, result.Session.ID)

				accounts, err := f.accountRepo.GetAllByUserID(ctx, result.User.ID)
				require.NoError(t, err)
				assert.Len(t, accounts, 2)

				linked, err := f.accountRepo.GetByProviderAndAccountID(ctx, "google", "google-linked")
				require.NoError(t, err)
				require.NotNil(t, linked)
				assert.Equal(t, result.User.ID, linked.UserID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackIntegrationFixture(t)
			provider, req := tt.setup(t, f)

			result, err := f.newUseCase(provider).Callback(context.Background(), req, nil, nil)

			require.NoError(t, err)
			tt.verify(t, f, result)
		})
	}
}
