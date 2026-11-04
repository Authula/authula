package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

type callbackFixture struct {
	useCase        *usecases.CallbackUseCase
	provider       *oauth2tests.MockProvider
	userService    *internaltests.MockUserService
	accountService *internaltests.MockAccountService
	sessionService *internaltests.MockSessionService
	tokenService   *internaltests.MockTokenService
	hmacKey        []byte
}

func newCallbackFixture(t *testing.T, provider *oauth2tests.MockProvider) *callbackFixture {
	t.Helper()

	hmacKey := services.DeriveOAuthHMACKey("test-secret")
	userService := &internaltests.MockUserService{}
	accountService := &internaltests.MockAccountService{}
	sessionService := &internaltests.MockSessionService{}
	tokenService := &internaltests.MockTokenService{}

	useCase := usecases.NewCallbackUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}},
		oauth2tests.NewRegistry(provider),
		&internaltests.MockLogger{},
		hmacKey,
		userService,
		accountService,
		sessionService,
		tokenService,
	)

	return &callbackFixture{
		useCase:        useCase,
		provider:       provider,
		userService:    userService,
		accountService: accountService,
		sessionService: sessionService,
		tokenService:   tokenService,
		hmacKey:        hmacKey,
	}
}

func (f *callbackFixture) request(t *testing.T, purpose services.StatePurpose, actorID string) *types.CallbackRequest {
	t.Helper()

	stateCookie, err := services.SignState(services.StatePayload{
		State:    "state-1",
		Provider: "google",
		UserID:   actorID,
		Purpose:  purpose,
	}, f.hmacKey)
	require.NoError(t, err)

	req := &types.CallbackRequest{
		ProviderID:  "google",
		Code:        "code-1",
		State:       "state-1",
		StateCookie: stateCookie,
		ActorID:     actorID,
	}
	if purpose == services.StatePurposeLink {
		req.SessionID = "session-1"
	}
	if f.provider.PKCE {
		verifierCookie, err := services.SignCookie("verifier-1", f.hmacKey)
		require.NoError(t, err)
		req.VerifierCookie = verifierCookie
	}
	return req
}

func (f *callbackFixture) assertExpectations(t *testing.T) {
	t.Helper()
	f.userService.AssertExpectations(t)
	f.accountService.AssertExpectations(t)
	f.sessionService.AssertExpectations(t)
	f.tokenService.AssertExpectations(t)
}

func newOAuthToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}
}

func TestCallbackUseCase_ExistingAccountUpdatesTokensForOwner(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		PKCE:         true,
		Token:        newOAuthToken(),
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-1",
			Email:             "user@example.com",
			EmailVerified:     true,
			Name:              "User",
		},
	}
	f := newCallbackFixture(t, provider)
	user := &models.User{ID: "user-1", Email: "user@example.com"}
	account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
	session := &models.Session{ID: "session-new", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
	f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
	f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
	f.tokenService.On("Generate").Return("plain-session-token", nil)
	f.tokenService.On("Hash", "plain-session-token").Return("hashed-session-token")
	f.sessionService.On("Create", mock.Anything, "user-1", "hashed-session-token", mock.Anything, mock.Anything, time.Hour).Return(session, nil)

	result, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLogin, ""), nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "user-1", result.User.ID)
	assert.Equal(t, "plain-session-token", result.SessionToken)
	assert.Equal(t, "code-1", provider.ExchangeCode)
	assert.Len(t, provider.ExchangeOpts, 1, "PKCE verifier must be passed to Exchange")
	f.assertExpectations(t)
}

func TestCallbackUseCase_RejectsAccountOwnedByAnotherUserDuringLink(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token:        newOAuthToken(),
		UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "provider@example.com", EmailVerified: true},
	}
	f := newCallbackFixture(t, provider)
	account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}

	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)

	_, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLink, "user-2"), nil, nil)

	require.ErrorIs(t, err, constants.ErrAccountLinkedToAnotherUser)
	f.assertExpectations(t)
}

func TestCallbackUseCase_AuthenticatedLinkReusesCurrentSession(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token:        newOAuthToken(),
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-2",
			Email:             "user@example.com",
			EmailVerified:     true,
		},
	}
	f := newCallbackFixture(t, provider)
	user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
	createdAccount := &models.Account{ID: "account-2", UserID: "user-1", ProviderID: "google", AccountID: "google-2"}
	session := &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-2").Return(nil, nil)
	f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
	f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, nil)
	f.accountService.On("CreateOAuth2", mock.Anything, "user-1", "google-2", "google", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(createdAccount, nil)
	f.sessionService.On("GetByID", mock.Anything, "session-1").Return(session, nil)

	result, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLink, "user-1"), nil, nil)

	require.NoError(t, err)
	assert.True(t, result.ReusedSession)
	assert.Equal(t, "session-1", result.Session.ID)
	assert.Empty(t, result.SessionToken)
	f.assertExpectations(t)
}

func TestCallbackUseCase_ValidationAndIdentityErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userInfo  *types.UserInfo
		wantError error
		setup     func(*callbackFixture)
	}{
		{
			name:      "missing email",
			userInfo:  &types.UserInfo{ProviderAccountID: "google-1"},
			wantError: constants.ErrEmailRequired,
		},
		{
			name:      "missing provider account id",
			userInfo:  &types.UserInfo{Email: "user@example.com", EmailVerified: true},
			wantError: constants.ErrProviderAccountIDRequired,
		},
		{
			name:      "nil provider account id",
			userInfo:  &types.UserInfo{ProviderAccountID: "<nil>", Email: "user@example.com", EmailVerified: true},
			wantError: constants.ErrProviderAccountIDRequired,
		},
		{
			name:     "unverified provider email does not auto-link",
			userInfo: &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: false},
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
			},
			wantError: constants.ErrEmailNotVerified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     tt.userInfo,
			}
			f := newCallbackFixture(t, provider)
			if tt.setup != nil {
				tt.setup(f)
			}

			_, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLogin, ""), nil, nil)

			require.ErrorIs(t, err, tt.wantError)
			f.assertExpectations(t)
		})
	}
}

func TestCallbackUseCase_IdentityIsAuthoritativeOverEmail(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token:        newOAuthToken(),
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-1",
			Email:             "changed@example.com",
			EmailVerified:     true,
		},
	}
	f := newCallbackFixture(t, provider)
	owner := &models.User{ID: "owner-1", Email: "original@example.com"}
	account := &models.Account{ID: "account-1", UserID: "owner-1", ProviderID: "google", AccountID: "google-1"}
	session := &models.Session{ID: "session-new", UserID: "owner-1", ExpiresAt: time.Now().Add(time.Hour)}

	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
	f.userService.On("GetByID", mock.Anything, "owner-1").Return(owner, nil)
	f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
	f.tokenService.On("Generate").Return("session-token", nil)
	f.tokenService.On("Hash", "session-token").Return("hashed")
	f.sessionService.On("Create", mock.Anything, "owner-1", "hashed", mock.Anything, mock.Anything, time.Hour).Return(session, nil)

	result, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLogin, ""), nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "owner-1", result.User.ID)
	f.assertExpectations(t)
}

func TestCallbackUseCase_AccountCreateFailureDeletesNewUser(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token:        newOAuthToken(),
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-1",
			Email:             "new@example.com",
			EmailVerified:     true,
		},
	}
	f := newCallbackFixture(t, provider)
	createdUser := &models.User{ID: "new-user", Email: "new@example.com"}

	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
	f.userService.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
	f.userService.On("Create", mock.Anything, "new", "new@example.com", true, mock.Anything, mock.Anything).Return(createdUser, nil)
	f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("unique constraint"))
	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
	f.userService.On("Delete", mock.Anything, "new-user").Return(nil)

	_, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLogin, ""), nil, nil)

	require.Error(t, err)
	f.assertExpectations(t)
}

func TestCallbackUseCase_UnverifiedNewEmailCreatesUnverifiedUser(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		Token:        newOAuthToken(),
		UserInfo: &types.UserInfo{
			ProviderAccountID: "google-1",
			Email:             "new@example.com",
			EmailVerified:     false,
			Picture:           "",
		},
	}
	f := newCallbackFixture(t, provider)
	createdUser := &models.User{ID: "new-user", Email: "new@example.com", EmailVerified: false}
	account := &models.Account{ID: "account-1", UserID: "new-user"}
	session := &models.Session{ID: "session-new", UserID: "new-user", ExpiresAt: time.Now().Add(time.Hour)}

	f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
	f.userService.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
	f.userService.On("Create", mock.Anything, "new", "new@example.com", false, (*string)(nil), mock.Anything).Return(createdUser, nil)
	f.accountService.On("CreateOAuth2", mock.Anything, "new-user", "google-1", "google", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
	f.tokenService.On("Generate").Return("session-token", nil)
	f.tokenService.On("Hash", "session-token").Return("hashed")
	f.sessionService.On("Create", mock.Anything, "new-user", "hashed", mock.Anything, mock.Anything, time.Hour).Return(session, nil)

	result, err := f.useCase.Callback(context.Background(), f.request(t, services.StatePurposeLogin, ""), nil, nil)

	require.NoError(t, err)
	assert.False(t, result.User.EmailVerified)
	f.assertExpectations(t)
}

func TestCallbackUseCase_PKCERequiresVerifier(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{
		ProviderName: "google",
		PKCE:         true,
		Token:        newOAuthToken(),
		UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
	}
	f := newCallbackFixture(t, provider)
	req := f.request(t, services.StatePurposeLogin, "")
	req.VerifierCookie = ""

	_, err := f.useCase.Callback(context.Background(), req, nil, nil)

	require.ErrorIs(t, err, constants.ErrPKCEVerifierMissing)
}

func TestCallbackUseCase_StateBindingErrors(t *testing.T) {
	t.Parallel()

	provider := &oauth2tests.MockProvider{ProviderName: "google", Token: newOAuthToken()}
	f := newCallbackFixture(t, provider)

	t.Run("provider mismatch", func(t *testing.T) {
		stateCookie, err := services.SignState(services.StatePayload{
			State:    "state-1",
			Provider: "github",
			Purpose:  services.StatePurposeLogin,
		}, f.hmacKey)
		require.NoError(t, err)

		req := f.request(t, services.StatePurposeLogin, "")
		req.StateCookie = stateCookie

		_, err = f.useCase.Callback(context.Background(), req, nil, nil)
		require.ErrorIs(t, err, constants.ErrStateProviderMismatch)
	})

	t.Run("actor mismatch for link", func(t *testing.T) {
		req := f.request(t, services.StatePurposeLink, "user-1")
		req.ActorID = "user-2"

		_, err := f.useCase.Callback(context.Background(), req, nil, nil)
		require.ErrorIs(t, err, constants.ErrUnauthorized)
	})
}
