package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

type callbackFixture struct {
	useCase        *callbackUseCase
	provider       *oauth2tests.MockProvider
	userService    *internaltests.MockUserService
	accountService *internaltests.MockAccountService
	sessionService *internaltests.MockSessionService
	tokenService   *internaltests.MockTokenService
	repo           *oauth2tests.FlowStateRepositoryFake
	logger         *internaltests.LevelRecordingLogger
	state          *types.OAuthFlowState

	verifierMissing bool
	verifierErr     error
}

func newCallbackFixture(t *testing.T, provider *oauth2tests.MockProvider) *callbackFixture {
	t.Helper()

	userService := &internaltests.MockUserService{}
	accountService := &internaltests.MockAccountService{}
	sessionService := &internaltests.MockSessionService{}
	tokenService := &internaltests.MockTokenService{}
	repo := oauth2tests.NewFlowStateRepositoryFake()
	logger := &internaltests.LevelRecordingLogger{}

	useCase := NewCallbackUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}},
		oauth2tests.NewRegistry(provider),
		logger,
		repo,
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
		repo:           repo,
		logger:         logger,
	}
}

func (f *callbackFixture) request(t *testing.T, purpose string, actorID string) *types.CallbackRequest {
	t.Helper()

	now := time.Now().UTC()
	state := &types.OAuthFlowState{
		ID:        "state-1",
		StateHash: util.HashWithSalt("state-1", "binding-1"),
		Provider:  "google",
		Purpose:   purpose,
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now,
	}
	if purpose == types.OAuthFlowPurposeLink {
		state.UserID = &actorID
	}
	if f.provider.PKCE && !f.verifierMissing {
		encrypted := "encrypted-verifier"
		state.CodeVerifierEncrypted = &encrypted
	}
	_, err := f.repo.Create(context.Background(), state)
	require.NoError(t, err)
	f.state = state

	req := &types.CallbackRequest{
		ProviderID: "google",
		Code:       "code-1",
		State:      "state-1",
		Binding:    "binding-1",
		ActorID:    actorID,
	}
	if purpose == types.OAuthFlowPurposeLink {
		req.SessionID = "session-1"
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

func TestCallbackUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		provider          *oauth2tests.MockProvider
		purpose           string
		actorID           string
		setup             func(*callbackFixture)
		mutateRequest     func(*callbackFixture, *types.CallbackRequest)
		wantErr           error
		wantAnyErr        bool
		wantUserID        string
		wantSessionToken  string
		wantReusedSession bool
		wantEmailVerified *bool
		wantRedirectTo    string
		wantRedirectError bool
		wantPKCE          bool
	}{
		{
			name: "existing account updates tokens for the owner",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				PKCE:         true,
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-1",
					Email:             "user@example.com",
					EmailVerified:     true,
					Name:              "User",
				},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com"}
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				session := &models.Session{ID: "session-new", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
				f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
				f.tokenService.On("Generate").Return("plain-session-token", nil)
				f.tokenService.On("Hash", "plain-session-token").Return("hashed-session-token")
				f.tokenService.On("Decrypt", "encrypted-verifier").Return("verifier-1", nil).Once()
				f.sessionService.On("Create", mock.Anything, "user-1", "hashed-session-token", mock.Anything, mock.Anything, time.Hour).Return(session, nil)
			},
			wantUserID:       "user-1",
			wantSessionToken: "plain-session-token",
			wantPKCE:         true,
		},
		{
			name: "authenticated link reuses the current session",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				PKCE:         true,
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-2",
					Email:             "user@example.com",
					EmailVerified:     true,
				},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
				createdAccount := &models.Account{ID: "account-2", UserID: "user-1", ProviderID: "google", AccountID: "google-2"}
				session := &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-2").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, "user-1", "google-2", "google", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(createdAccount, nil)
				f.tokenService.On("Decrypt", "encrypted-verifier").Return("verifier-1", nil).Once()
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(session, nil)
			},
			wantUserID:        "user-1",
			wantReusedSession: true,
			wantPKCE:          true,
		},
		{
			name: "link marks a matching verified email as verified",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-3",
					Email:             "user@example.com",
					EmailVerified:     true,
				},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: false}
				verifiedUser := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
				createdAccount := &models.Account{ID: "account-3", UserID: "user-1", ProviderID: "google", AccountID: "google-3"}
				session := &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-3").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, "user-1", "google-3", "google", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(createdAccount, nil)
				f.userService.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
					return u.Email == "user@example.com" && u.EmailVerified
				})).Return(verifiedUser, nil)
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(session, nil)
			},
			wantUserID:        "user-1",
			wantReusedSession: true,
			wantEmailVerified: new(true),
		},
		{
			name: "identity is authoritative over the provider email",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-1",
					Email:             "changed@example.com",
					EmailVerified:     true,
				},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				owner := &models.User{ID: "owner-1", Email: "original@example.com"}
				account := &models.Account{ID: "account-1", UserID: "owner-1", ProviderID: "google", AccountID: "google-1"}
				session := &models.Session{ID: "session-new", UserID: "owner-1", ExpiresAt: time.Now().Add(time.Hour)}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
				f.userService.On("GetByID", mock.Anything, "owner-1").Return(owner, nil)
				f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
				f.tokenService.On("Generate").Return("session-token", nil)
				f.tokenService.On("Hash", "session-token").Return("hashed")
				f.sessionService.On("Create", mock.Anything, "owner-1", "hashed", mock.Anything, mock.Anything, time.Hour).Return(session, nil)
			},
			wantUserID:       "owner-1",
			wantSessionToken: "session-token",
		},
		{
			name: "unverified new email creates an unverified user",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-1",
					Email:             "new@example.com",
					EmailVerified:     false,
				},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
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
			},
			wantUserID:        "new-user",
			wantSessionToken:  "session-token",
			wantEmailVerified: new(false),
		},
		{
			name: "account create race resolves to the same owner",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-1",
					Email:             "user@example.com",
					EmailVerified:     true,
				},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				session := &models.Session{ID: "session-new", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, "user-1", "google-1", "google", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("unique constraint"))
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil).Once()
				f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
				f.tokenService.On("Generate").Return("session-token", nil)
				f.tokenService.On("Hash", "session-token").Return("hashed")
				f.sessionService.On("Create", mock.Anything, "user-1", "hashed", mock.Anything, mock.Anything, time.Hour).Return(session, nil)
			},
			wantUserID:       "user-1",
			wantSessionToken: "session-token",
		},
		{
			name: "rejects an account owned by another user during link",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "provider@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-2",
			setup: func(f *callbackFixture) {
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
			},
			wantErr: constants.ErrAccountLinkedToAnotherUser,
		},
		{
			name: "missing email",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1"},
			},
			purpose: types.OAuthFlowPurposeLogin,
			wantErr: constants.ErrEmailRequired,
		},
		{
			name: "missing provider account id",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			wantErr: constants.ErrProviderAccountIDRequired,
		},
		{
			name: "nil provider account id",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "<nil>", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			wantErr: constants.ErrProviderAccountIDRequired,
		},
		{
			name: "unverified provider email does not auto-link",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: false},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
			},
			wantErr: constants.ErrEmailNotVerified,
		},
		{
			name: "account create failure deletes the new user",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-1",
					Email:             "new@example.com",
					EmailVerified:     true,
				},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				createdUser := &models.User{ID: "new-user", Email: "new@example.com"}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
				f.userService.On("Create", mock.Anything, "new", "new@example.com", true, mock.Anything, mock.Anything).Return(createdUser, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("unique constraint"))
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("Delete", mock.Anything, "new-user").Return(nil)
			},
			wantAnyErr: true,
		},
		{
			name: "provider denied access",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.Error = "access_denied"
			},
			wantErr: constants.ErrAccessDenied,
		},
		{
			name: "provider error redirects to the stored redirect",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(f *callbackFixture, req *types.CallbackRequest) {
				req.Error = "access_denied"
				req.ErrorDescription = "user cancelled"
				f.state.RedirectTo = new("/welcome")
			},
			wantRedirectError: true,
			wantRedirectTo:    "/welcome",
		},
		{
			name: "provider error with an untrusted redirect returns access denied",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(f *callbackFixture, req *types.CallbackRequest) {
				req.Error = "access_denied"
				f.state.RedirectTo = new("https://evil.example/callback")
			},
			wantErr: constants.ErrAccessDenied,
		},
		{
			name: "missing state",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.State = ""
			},
			wantErr: constants.ErrInvalidState,
		},
		{
			name: "missing binding",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.Binding = ""
			},
			wantErr: constants.ErrInvalidState,
		},
		{
			name: "unknown state",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.State = "other-state"
			},
			wantErr: constants.ErrInvalidState,
		},
		{
			name: "expired state",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(f *callbackFixture, _ *types.CallbackRequest) {
				f.state.ExpiresAt = time.Now().UTC().Add(-time.Minute)
			},
			wantErr: constants.ErrInvalidState,
		},
		{
			name: "already consumed state",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(f *callbackFixture, _ *types.CallbackRequest) {
				f.state.ConsumedAt = new(time.Now().UTC())
			},
			wantErr: constants.ErrInvalidState,
		},
		{
			name: "unknown purpose",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(f *callbackFixture, _ *types.CallbackRequest) {
				f.state.Purpose = "invalid"
			},
			wantErr: constants.ErrInvalidState,
		},
		{
			name: "login flow rejected when an actor is present",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			actorID: "user-1",
			wantErr: constants.ErrUnauthorized,
		},
		{
			name: "missing authorization code",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.Code = ""
			},
			wantErr: constants.ErrInvalidCode,
		},
		{
			name: "provider exchange failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				ExchangeErr:  errors.New("exchange failed"),
			},
			purpose: types.OAuthFlowPurposeLogin,
			wantErr: constants.ErrProviderFailure,
		},
		{
			name: "provider user info failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfoErr:  errors.New("user info failed"),
			},
			purpose: types.OAuthFlowPurposeLogin,
			wantErr: constants.ErrProviderFailure,
		},
		{
			name: "empty access token",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        &oauth2.Token{},
			},
			purpose: types.OAuthFlowPurposeLogin,
			wantErr: constants.ErrProviderFailure,
		},
		{
			name: "pkce decrypt failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				PKCE:         true,
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.verifierErr = errors.New("decrypt failed")
				f.tokenService.On("Decrypt", "encrypted-verifier").Return("", f.verifierErr).Once()
			},
			wantErr: constants.ErrPKCEVerifierMissing,
		},
		{
			name: "account lookup failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, errors.New("lookup failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "account owner lookup failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(nil, errors.New("load failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "account owner is missing",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(nil, nil)
			},
			wantErr: constants.ErrAccountConflict,
		},
		{
			name: "email lookup failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, errors.New("lookup failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "user creation reports email already in use",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, nil)
				f.userService.On("Create", mock.Anything, mock.Anything, "user@example.com", true, mock.Anything, mock.Anything).Return(nil, coreservices.ErrEmailInUse)
			},
			wantErr: constants.ErrEmailInUse,
		},
		{
			name: "user creation failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, nil)
				f.userService.On("Create", mock.Anything, mock.Anything, "user@example.com", true, mock.Anything, mock.Anything).Return(nil, errors.New("create failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "link user lookup failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(nil, errors.New("load failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "link user is missing",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(nil, nil)
			},
			wantErr: constants.ErrUnauthorized,
		},
		{
			name: "link account check failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, errors.New("lookup failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "provider already linked to the user",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(&models.Account{ID: "account-existing", UserID: "user-1", ProviderID: "google"}, nil)
			},
			wantErr: constants.ErrAccountConflict,
		},
		{
			name: "token update failure",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "token update rejects an expired token",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, coreservices.ErrTokenExpired)
			},
			wantErr: constants.ErrTokenExpired,
		},
		{
			name: "account create race belongs to another user",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("unique constraint"))
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(&models.Account{ID: "account-1", UserID: "user-2", ProviderID: "google", AccountID: "google-1"}, nil).Once()
			},
			wantErr: constants.ErrAccountLinkedToAnotherUser,
		},
		{
			name: "account create reports another owner",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, coreservices.ErrAccountLinkedToAnotherUser)
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
			},
			wantErr: constants.ErrAccountLinkedToAnotherUser,
		},
		{
			name: "account create reports an expired token",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, coreservices.ErrTokenExpired)
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
			},
			wantErr: constants.ErrTokenExpired,
		},
		{
			name: "link account create failure does not delete the user",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
				f.userService.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("create failed"))
				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil).Once()
			},
			wantAnyErr: true,
		},
		{
			name: "link verification update failure leaves the user unverified",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: false}
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
				session := &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
				f.userService.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(session, nil)
			},
			wantUserID:        "user-1",
			wantReusedSession: true,
			wantEmailVerified: new(false),
		},
		{
			name: "link fails when the current session is missing",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
				f.accountService.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
			},
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.SessionID = ""
			},
			wantErr: constants.ErrUnauthorized,
		},
		{
			name: "login fails when session creation fails",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
				account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}

				f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, nil)
				f.userService.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
				f.accountService.On("CreateOAuth2", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
				f.tokenService.On("Generate").Return(nil, errors.New("generate failed"))
			},
			wantAnyErr: true,
		},
		{
			name: "pkce requires a verifier",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				PKCE:         true,
				Token:        newOAuthToken(),
				UserInfo:     &types.UserInfo{ProviderAccountID: "google-1", Email: "user@example.com", EmailVerified: true},
			},
			purpose: types.OAuthFlowPurposeLogin,
			setup: func(f *callbackFixture) {
				f.verifierMissing = true
			},
			wantErr: constants.ErrPKCEVerifierMissing,
		},
		{
			name: "state provider mismatch",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
			},
			purpose: types.OAuthFlowPurposeLogin,
			mutateRequest: func(f *callbackFixture, _ *types.CallbackRequest) {
				f.state.Provider = "github"
			},
			wantErr: constants.ErrStateProviderMismatch,
		},
		{
			name: "actor mismatch for link",
			provider: &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
			},
			purpose: types.OAuthFlowPurposeLink,
			actorID: "user-1",
			mutateRequest: func(_ *callbackFixture, req *types.CallbackRequest) {
				req.ActorID = "user-2"
			},
			wantErr: constants.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, tt.provider)
			if tt.setup != nil {
				tt.setup(f)
			}

			req := f.request(t, tt.purpose, tt.actorID)
			if tt.mutateRequest != nil {
				tt.mutateRequest(f, req)
			}

			result, err := f.useCase.Callback(context.Background(), req, nil, nil)

			if tt.wantRedirectError {
				require.Error(t, err)
				require.Nil(t, result)
				redirectErr := new(RedirectError)
				require.ErrorAs(t, err, &redirectErr)
				assert.Equal(t, tt.wantRedirectTo, redirectErr.RedirectTo)
				assert.Equal(t, "access_denied", redirectErr.Code)
				f.assertExpectations(t)
				return
			}
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				f.assertExpectations(t)
				return
			}
			if tt.wantAnyErr {
				require.Error(t, err)
				assert.Nil(t, result)
				f.assertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.User)
			require.NotNil(t, result.Session)
			assert.Equal(t, tt.wantUserID, result.User.ID)
			assert.Equal(t, tt.wantSessionToken, result.SessionToken)
			assert.Equal(t, tt.wantReusedSession, result.ReusedSession)
			assert.Equal(t, tt.wantRedirectTo, result.RedirectTo)
			assert.Equal(t, "code-1", tt.provider.ExchangeCode)

			if tt.wantPKCE {
				assert.Len(t, tt.provider.ExchangeOpts, 1, "PKCE verifier must be passed to Exchange")
				f.tokenService.AssertNumberOfCalls(t, "Decrypt", 1)
			} else {
				assert.Empty(t, tt.provider.ExchangeOpts)
				f.tokenService.AssertNumberOfCalls(t, "Decrypt", 0)
			}
			if tt.wantEmailVerified != nil {
				assert.Equal(t, *tt.wantEmailVerified, result.User.EmailVerified)
			}

			f.assertExpectations(t)
		})
	}
}

func TestCallbackUseCase_RedirectValidationIsApplied(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		trustedOrigins []string
		redirectTo     *string
		wantRedirectTo string
		wantWarning    string
	}{
		{
			name:           "relative redirect is returned",
			redirectTo:     new("/welcome"),
			wantRedirectTo: "/welcome",
		},
		{
			name:           "trusted absolute redirect is returned",
			trustedOrigins: []string{"https://app.example.com"},
			redirectTo:     new("https://app.example.com/welcome"),
			wantRedirectTo: "https://app.example.com/welcome",
		},
		{
			name:           "untrusted redirect is dropped and logged",
			trustedOrigins: []string{"https://app.example.com"},
			redirectTo:     new("https://evil.example/callback"),
			wantWarning:    "oauth redirect_to failed validation; redirect suppressed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &oauth2tests.MockProvider{
				ProviderName: "google",
				Token:        newOAuthToken(),
				UserInfo: &types.UserInfo{
					ProviderAccountID: "google-1",
					Email:             "user@example.com",
					EmailVerified:     true,
				},
			}
			f := newCallbackFixture(t, provider)
			f.useCase.GlobalConfig.Security.TrustedOrigins = tt.trustedOrigins

			user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}
			account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
			session := &models.Session{ID: "session-new", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}

			f.accountService.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
			f.userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)
			f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", "new-access-token", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
			f.tokenService.On("Generate").Return("session-token", nil)
			f.tokenService.On("Hash", "session-token").Return("hashed")
			f.sessionService.On("Create", mock.Anything, "user-1", "hashed", mock.Anything, mock.Anything, time.Hour).Return(session, nil)

			req := f.request(t, types.OAuthFlowPurposeLogin, "")
			f.state.RedirectTo = tt.redirectTo

			result, err := f.useCase.Callback(context.Background(), req, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.User)
			require.NotNil(t, result.Session)
			assert.Equal(t, tt.wantRedirectTo, result.RedirectTo)

			if tt.wantWarning != "" {
				assert.Contains(t, f.logger.Messages(), tt.wantWarning)
			} else {
				assert.NotContains(t, f.logger.Messages(), "oauth redirect_to failed validation; redirect suppressed")
			}

			f.assertExpectations(t)
		})
	}
}

func TestCallbackUseCase_CurrentSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		sessionID     string
		actorID       string
		setup         func(*callbackFixture)
		wantErr       error
		wantAnyErr    bool
		wantSessionID string
	}{
		{
			name:    "missing session id",
			actorID: "user-1",
			wantErr: constants.ErrUnauthorized,
		},
		{
			name:      "session lookup failure",
			sessionID: "session-1",
			actorID:   "user-1",
			setup: func(f *callbackFixture) {
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(nil, errors.New("lookup failed"))
			},
			wantAnyErr: true,
		},
		{
			name:      "session not found",
			sessionID: "session-1",
			actorID:   "user-1",
			setup: func(f *callbackFixture) {
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(nil, nil)
			},
			wantErr: constants.ErrSessionNotFound,
		},
		{
			name:      "session belongs to another user",
			sessionID: "session-1",
			actorID:   "user-1",
			setup: func(f *callbackFixture) {
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(&models.Session{
					ID: "session-1", UserID: "user-2", ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			wantErr: constants.ErrSessionNotFound,
		},
		{
			name:      "expired session",
			sessionID: "session-1",
			actorID:   "user-1",
			setup: func(f *callbackFixture) {
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(&models.Session{
					ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			wantErr: constants.ErrSessionNotFound,
		},
		{
			name:      "valid session",
			sessionID: "session-1",
			actorID:   "user-1",
			setup: func(f *callbackFixture) {
				f.sessionService.On("GetByID", mock.Anything, "session-1").Return(&models.Session{
					ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			wantSessionID: "session-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, &oauth2tests.MockProvider{ProviderName: "google"})
			if tt.setup != nil {
				tt.setup(f)
			}

			session, err := f.useCase.currentSession(context.Background(), &types.CallbackRequest{
				SessionID: tt.sessionID,
				ActorID:   tt.actorID,
			})

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, session)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Nil(t, session)
			default:
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tt.wantSessionID, session.ID)
			}
			f.assertExpectations(t)
		})
	}
}

func TestCallbackUseCase_CreateSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setup         func(*callbackFixture)
		wantErr       bool
		wantToken     string
		wantSessionID string
	}{
		{
			name: "token generation failure",
			setup: func(f *callbackFixture) {
				f.tokenService.On("Generate").Return(nil, errors.New("generate failed"))
			},
			wantErr: true,
		},
		{
			name: "session creation failure",
			setup: func(f *callbackFixture) {
				f.tokenService.On("Generate").Return("session-token", nil)
				f.tokenService.On("Hash", "session-token").Return("hashed")
				f.sessionService.On("Create", mock.Anything, "user-1", "hashed", mock.Anything, mock.Anything, time.Hour).Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(f *callbackFixture) {
				f.tokenService.On("Generate").Return("session-token", nil)
				f.tokenService.On("Hash", "session-token").Return("hashed")
				f.sessionService.On("Create", mock.Anything, "user-1", "hashed", mock.Anything, mock.Anything, time.Hour).Return(&models.Session{
					ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			wantToken:     "session-token",
			wantSessionID: "session-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, &oauth2tests.MockProvider{ProviderName: "google"})
			tt.setup(f)

			token, session, err := f.useCase.createSession(context.Background(), "user-1", nil, nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, token)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
				require.NotNil(t, session)
				assert.Equal(t, tt.wantSessionID, session.ID)
			}
			f.assertExpectations(t)
		})
	}
}

func TestCallbackUseCase_Redirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state *types.OAuthFlowState
		want  string
	}{
		{
			name: "nil state",
		},
		{
			name:  "missing redirect",
			state: &types.OAuthFlowState{},
		},
		{
			name:  "relative redirect",
			state: &types.OAuthFlowState{RedirectTo: new("/welcome")},
			want:  "/welcome",
		},
		{
			name:  "trusted absolute redirect",
			state: &types.OAuthFlowState{RedirectTo: new("https://app.example.com/welcome")},
			want:  "https://app.example.com/welcome",
		},
		{
			name:  "untrusted redirect",
			state: &types.OAuthFlowState{RedirectTo: new("https://evil.example/callback")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, &oauth2tests.MockProvider{ProviderName: "google"})
			f.useCase.GlobalConfig.Security.TrustedOrigins = []string{"https://app.example.com"}

			got := f.useCase.redirectTo(tt.state)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		info    *types.UserInfo
		wantErr error
	}{
		{name: "nil info", info: nil, wantErr: constants.ErrProviderFailure},
		{name: "empty provider id", info: &types.UserInfo{Email: "user@example.com"}, wantErr: constants.ErrProviderAccountIDRequired},
		{name: "nil-like provider id", info: &types.UserInfo{ProviderAccountID: "<nil>", Email: "user@example.com"}, wantErr: constants.ErrProviderAccountIDRequired},
		{name: "null provider id", info: &types.UserInfo{ProviderAccountID: "null", Email: "user@example.com"}, wantErr: constants.ErrProviderAccountIDRequired},
		{name: "empty email", info: &types.UserInfo{ProviderAccountID: "provider-1"}, wantErr: constants.ErrEmailRequired},
		{name: "valid", info: &types.UserInfo{ProviderAccountID: "provider-1", Email: "user@example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateUserInfo(tt.info)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFallbackName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info *types.UserInfo
		want string
	}{
		{name: "provider name", info: &types.UserInfo{Name: "  User Name  ", Email: "user@example.com"}, want: "User Name"},
		{name: "email local part fallback", info: &types.UserInfo{Email: "user@example.com"}, want: "user"},
		{name: "generic fallback", info: &types.UserInfo{Email: "invalid"}, want: "User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fallbackName(tt.info))
		})
	}
}

func TestCallbackTokenHelpers(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour)

	tests := []struct {
		name           string
		expiry         time.Time
		refreshToken   string
		image          string
		wantExpiry     *time.Time
		wantRefreshNil bool
		wantImageNil   bool
	}{
		{name: "zero expiry", expiry: time.Time{}, refreshToken: "refresh", image: "image", wantRefreshNil: false, wantImageNil: false},
		{name: "non-zero expiry and empty optional values", expiry: future, refreshToken: "  ", image: "  ", wantExpiry: &future, wantRefreshNil: true, wantImageNil: true},
		{name: "non-empty optional values", expiry: future, refreshToken: "refresh", image: "image", wantExpiry: &future},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantExpiry, tokenExpiry(&oauth2.Token{Expiry: tt.expiry}))
			if tt.wantRefreshNil {
				assert.Nil(t, optionalToken(tt.refreshToken))
			} else {
				require.NotNil(t, optionalToken(tt.refreshToken))
				assert.Equal(t, tt.refreshToken, *optionalToken(tt.refreshToken))
			}
			if tt.wantImageNil {
				assert.Nil(t, imagePointer(tt.image))
			} else {
				require.NotNil(t, imagePointer(tt.image))
				assert.Equal(t, tt.image, *imagePointer(tt.image))
			}
		})
	}
}

func TestTokenScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token *oauth2.Token
		want  *string
	}{
		{name: "nil token", token: nil},
		{name: "missing scope", token: &oauth2.Token{}},
		{name: "empty scope", token: (&oauth2.Token{}).WithExtra(map[string]any{"scope": "   "})},
		{name: "string scope", token: (&oauth2.Token{}).WithExtra(map[string]any{"scope": "openid email"}), want: new("openid email")},
		{name: "string slice scope", token: (&oauth2.Token{}).WithExtra(map[string]any{"scope": []string{"openid", "email"}}), want: new("openid email")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tokenScope(tt.token)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func TestCallbackUseCase_DeleteOrphanUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		deleteErr error
		wantLogs  int
	}{
		{name: "success does not log", deleteErr: nil, wantLogs: 0},
		{name: "failure is logged", deleteErr: errors.New("delete failed"), wantLogs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, &oauth2tests.MockProvider{ProviderName: "google"})
			logger := &internaltests.RecordingLogger{}
			f.useCase.Logger = logger
			f.userService.On("Delete", mock.Anything, "user-1").Return(tt.deleteErr)

			f.useCase.deleteOrphanUser(context.Background(), "user-1")

			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			f.assertExpectations(t)
		})
	}
}

func TestCallbackUseCase_PersistAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		existingUserID string
		token          *oauth2.Token
		setup          func(*callbackFixture, *models.Account)
		wantErr        error
	}{
		{
			name:           "rejects a mismatched owner",
			existingUserID: "user-2",
			wantErr:        constants.ErrAccountLinkedToAnotherUser,
		},
		{
			name:           "updates tokens for the matching owner",
			existingUserID: "user-1",
			setup: func(f *callbackFixture, account *models.Account) {
				f.accountService.On("UpdateOAuth2Tokens", mock.Anything, "account-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(account, nil)
			},
		},
		{
			name:           "persists the provider-granted scope",
			existingUserID: "user-1",
			token:          newOAuthToken().WithExtra(map[string]any{"scope": "openid email"}),
			setup: func(f *callbackFixture, account *models.Account) {
				f.accountService.On(
					"UpdateOAuth2Tokens",
					mock.Anything,
					"account-1",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(scope *string) bool { return scope != nil && *scope == "openid email" }),
				).Return(account, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, &oauth2tests.MockProvider{ProviderName: "google"})
			account := &models.Account{ID: "account-1", UserID: tt.existingUserID, ProviderID: "google", AccountID: "google-1"}
			if tt.setup != nil {
				tt.setup(f, account)
			}

			token := tt.token
			if token == nil {
				token = newOAuthToken()
			}

			err := f.useCase.persistAccount(
				context.Background(),
				"google",
				&models.User{ID: "user-1"},
				&types.UserInfo{ProviderAccountID: "google-1"},
				token,
				account,
				false,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			f.assertExpectations(t)
		})
	}
}
