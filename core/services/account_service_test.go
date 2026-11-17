package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
)

func newAccountServiceForTest() (services.AccountService, *internaltests.MockAccountRepository, *internaltests.MockTokenRepository) {
	return newAccountServiceWithHooksForTest(nil, &internaltests.MockLogger{})
}

func newAccountServiceWithHooksForTest(hooks *models.CoreServiceHooksConfig, logger models.Logger) (services.AccountService, *internaltests.MockAccountRepository, *internaltests.MockTokenRepository) {
	repo := new(internaltests.MockAccountRepository)
	tokenRepo := new(internaltests.MockTokenRepository)
	svc := coreservices.NewAccountService(nil, repo, tokenRepo, hooks, logger)
	return svc, repo, tokenRepo
}

func TestAccountService_CreateOAuth2(t *testing.T) {
	t.Parallel()

	storedRefresh := "encrypted-old-refresh"
	existingAccount := &models.Account{
		ID:           "account-1",
		UserID:       "user-1",
		ProviderID:   "google",
		AccountID:    "google-1",
		RefreshToken: &storedRefresh,
	}

	tests := []struct {
		name           string
		callerUserID   string
		existing       *models.Account
		refreshToken   *string
		encryptRefresh *string
		wantRefresh    *string
		wantErr        error
		expectUpdate   bool
	}{
		{
			name:         "rejects an account owned by another user",
			callerUserID: "user-2",
			existing:     existingAccount,
			wantErr:      coreservices.ErrAccountLinkedToAnotherUser,
		},
		{
			name:         "preserves stored refresh token when the new one is empty",
			callerUserID: "user-1",
			existing:     existingAccount,
			refreshToken: new(""),
			wantRefresh:  &storedRefresh,
			expectUpdate: true,
		},
		{
			name:           "stores and encrypts a new refresh token",
			callerUserID:   "user-1",
			existing:       existingAccount,
			refreshToken:   new("new-refresh"),
			encryptRefresh: new("encrypted-new-refresh"),
			wantRefresh:    new("encrypted-new-refresh"),
			expectUpdate:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenRepo := newAccountServiceForTest()
			existing := tt.existing
			if existing != nil {
				clone := *existing
				existing = &clone
			}

			tokenRepo.On("Encrypt", "access").Return("encrypted-access", nil)
			if tt.encryptRefresh != nil {
				tokenRepo.On("Encrypt", "new-refresh").Return(*tt.encryptRefresh, nil)
			}
			repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(existing, nil)
			if tt.expectUpdate {
				repo.On("Update", mock.Anything, mock.Anything).Return(existing, nil)
			}

			updated, err := svc.CreateOAuth2(
				context.Background(),
				tt.callerUserID,
				"google-1",
				"google",
				"access",
				tt.refreshToken,
				nil,
				nil,
				nil,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			} else {
				require.NoError(t, err)
				require.NotNil(t, updated)
				assert.Equal(t, tt.wantRefresh, updated.RefreshToken)
			}

			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_UpdateOAuth2Tokens(t *testing.T) {
	t.Parallel()

	now := time.Now()
	future := now.Add(time.Hour)
	futureRefresh := now.Add(48 * time.Hour)
	newScope := "new-scope"
	storedRefresh := "encrypted-old-refresh"
	storedRefreshExpiry := now.Add(48 * time.Hour)
	storedScope := "old-scope"
	lookupErr := errors.New("account lookup failed")

	accessEncryptErr := errors.New("access token encryption failed")
	refreshEncryptErr := errors.New("refresh token encryption failed")

	tests := []struct {
		name                string
		account             *models.Account
		accessToken         string
		refreshToken        *string
		accessExpiry        *time.Time
		refreshExpiry       *time.Time
		scope               *string
		encryptAccess       string
		encryptAccessErr    error
		encryptRefresh      *string
		encryptRefreshErr   error
		wantRefresh         *string
		wantRefreshExpiry   *time.Time
		wantScope           *string
		getByIDErr          error
		wantErr             error
		expectGetByID       bool
		expectEncryptAccess bool
	}{
		{
			name: "updates encrypted access token scoped to the account",
			account: &models.Account{
				ID:                    "account-1",
				UserID:                "user-1",
				RefreshToken:          &storedRefresh,
				RefreshTokenExpiresAt: &storedRefreshExpiry,
				Scope:                 &storedScope,
			},
			accessToken:         "new-access",
			accessExpiry:        &future,
			encryptAccess:       "encrypted-new-access",
			wantRefresh:         &storedRefresh,
			wantRefreshExpiry:   &storedRefreshExpiry,
			wantScope:           &storedScope,
			expectGetByID:       true,
			expectEncryptAccess: true,
		},
		{
			name: "empty refresh token preserves stored refresh token",
			account: &models.Account{
				ID:           "account-1",
				UserID:       "user-1",
				RefreshToken: &storedRefresh,
			},
			accessToken:         "new-access",
			refreshToken:        new(""),
			encryptAccess:       "encrypted-new-access",
			wantRefresh:         &storedRefresh,
			expectGetByID:       true,
			expectEncryptAccess: true,
		},
		{
			name: "new refresh token is encrypted",
			account: &models.Account{
				ID:     "account-1",
				UserID: "user-1",
			},
			accessToken:         "new-access",
			refreshToken:        new("new-refresh"),
			encryptAccess:       "encrypted-new-access",
			encryptRefresh:      new("encrypted-new-refresh"),
			wantRefresh:         new("encrypted-new-refresh"),
			expectGetByID:       true,
			expectEncryptAccess: true,
		},
		{
			name: "applies refresh token expiry and scope when provided",
			account: &models.Account{
				ID:     "account-1",
				UserID: "user-1",
			},
			accessToken:         "new-access",
			accessExpiry:        &future,
			refreshExpiry:       &futureRefresh,
			scope:               &newScope,
			encryptAccess:       "encrypted-new-access",
			wantRefreshExpiry:   &futureRefresh,
			wantScope:           &newScope,
			expectGetByID:       true,
			expectEncryptAccess: true,
		},
		{
			name:          "returns account lookup errors",
			accessToken:   "new-access",
			getByIDErr:    lookupErr,
			wantErr:       lookupErr,
			expectGetByID: true,
		},
		{
			name:          "rejects expired token",
			accessToken:   "new-access",
			accessExpiry:  new(now.Add(-time.Minute)),
			wantErr:       coreservices.ErrTokenExpired,
			expectGetByID: false,
		},
		{
			name:          "rejects an empty access token",
			accessToken:   "",
			wantErr:       coreservices.ErrTokenExpired,
			expectGetByID: false,
		},
		{
			name:          "missing account",
			accessToken:   "new-access",
			account:       nil,
			wantErr:       coreservices.ErrAccountNotFound,
			expectGetByID: true,
		},
		{
			name:                "returns access token encryption errors",
			accessToken:         "new-access",
			account:             &models.Account{ID: "account-1", UserID: "user-1"},
			encryptAccessErr:    accessEncryptErr,
			wantErr:             accessEncryptErr,
			expectGetByID:       true,
			expectEncryptAccess: true,
		},
		{
			name:                "returns refresh token encryption errors",
			accessToken:         "new-access",
			refreshToken:        new("new-refresh"),
			account:             &models.Account{ID: "account-1", UserID: "user-1"},
			encryptAccess:       "encrypted-new-access",
			encryptRefreshErr:   refreshEncryptErr,
			wantErr:             refreshEncryptErr,
			expectGetByID:       true,
			expectEncryptAccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenRepo := newAccountServiceForTest()
			if tt.expectGetByID {
				repo.On("GetByID", mock.Anything, "account-1").Return(tt.account, tt.getByIDErr)
			}
			if tt.expectEncryptAccess {
				tokenRepo.On("Encrypt", tt.accessToken).Return(tt.encryptAccess, tt.encryptAccessErr)
			}
			if tt.encryptRefresh != nil || tt.encryptRefreshErr != nil {
				encryptedRefresh := ""
				if tt.encryptRefresh != nil {
					encryptedRefresh = *tt.encryptRefresh
				}
				tokenRepo.On("Encrypt", "new-refresh").Return(encryptedRefresh, tt.encryptRefreshErr)
			}
			if tt.wantErr == nil && tt.expectGetByID && tt.account != nil {
				repo.On("Update", mock.Anything, mock.Anything).Return(tt.account, nil)
			}

			updated, err := svc.UpdateOAuth2Tokens(
				context.Background(),
				"account-1",
				tt.accessToken,
				tt.refreshToken,
				tt.accessExpiry,
				tt.refreshExpiry,
				tt.scope,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, updated)
				repo.AssertExpectations(t)
				tokenRepo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, tt.wantRefresh, updated.RefreshToken)
			assert.Equal(t, tt.wantRefreshExpiry, updated.RefreshTokenExpiresAt)
			assert.Equal(t, tt.wantScope, updated.Scope)
			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_ReadDelegation(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("account repository unavailable")
	account := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}

	tests := []struct {
		name    string
		setup   func(*internaltests.MockAccountRepository)
		call    func(context.Context, services.AccountService) (*models.Account, error)
		want    *models.Account
		wantErr error
	}{
		{
			name: "GetByUserID returns the repository account",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByUserID", mock.Anything, "user-1").Return(account, nil)
			},
			call: func(ctx context.Context, svc services.AccountService) (*models.Account, error) {
				return svc.GetByUserID(ctx, "user-1")
			},
			want: account,
		},
		{
			name: "GetByUserID returns repository errors",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByUserID", mock.Anything, "user-1").Return(nil, repoErr)
			},
			call: func(ctx context.Context, svc services.AccountService) (*models.Account, error) {
				return svc.GetByUserID(ctx, "user-1")
			},
			wantErr: repoErr,
		},
		{
			name: "GetByUserIDAndProvider passes arguments through unchanged",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(account, nil)
			},
			call: func(ctx context.Context, svc services.AccountService) (*models.Account, error) {
				return svc.GetByUserIDAndProvider(ctx, "user-1", "google")
			},
			want: account,
		},
		{
			name: "GetByUserIDAndProvider returns repository errors",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByUserIDAndProvider", mock.Anything, "user-1", "google").Return(nil, repoErr)
			},
			call: func(ctx context.Context, svc services.AccountService) (*models.Account, error) {
				return svc.GetByUserIDAndProvider(ctx, "user-1", "google")
			},
			wantErr: repoErr,
		},
		{
			name: "GetByProviderAndAccountID passes arguments through unchanged",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(account, nil)
			},
			call: func(ctx context.Context, svc services.AccountService) (*models.Account, error) {
				return svc.GetByProviderAndAccountID(ctx, "google", "google-1")
			},
			want: account,
		},
		{
			name: "GetByProviderAndAccountID returns repository errors",
			setup: func(repo *internaltests.MockAccountRepository) {
				repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, repoErr)
			},
			call: func(ctx context.Context, svc services.AccountService) (*models.Account, error) {
				return svc.GetByProviderAndAccountID(ctx, "google", "google-1")
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenRepo := newAccountServiceForTest()
			tt.setup(repo)

			got, err := tt.call(context.Background(), svc)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Same(t, tt.want, got)
			}

			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_Create(t *testing.T) {
	t.Parallel()

	createErr := errors.New("create failed")
	beforeHookErr := errors.New("before create hook failed")
	afterHookErr := errors.New("after create hook failed")
	password := "hashed-password"

	tests := []struct {
		name          string
		accountID     string
		providerID    string
		password      *string
		beforeHook    models.ServiceHook[models.Account]
		afterHook     models.ServiceHook[models.Account]
		createErr     error
		wantAccountID string
		wantErr       error
		wantLogs      int
		expectCreate  bool
	}{
		{
			name:          "normalizes email account IDs",
			accountID:     "  User@Example.COM  ",
			providerID:    models.AuthProviderEmail.String(),
			password:      &password,
			wantAccountID: "user@example.com",
			expectCreate:  true,
		},
		{
			name:          "keeps account IDs for other providers unchanged",
			accountID:     "  Google-Account  ",
			providerID:    "google",
			wantAccountID: "  Google-Account  ",
			expectCreate:  true,
		},
		{
			name:       "returns before create hook errors",
			accountID:  "acc-1",
			providerID: "google",
			beforeHook: func(a *models.Account) error { return beforeHookErr },
			wantErr:    beforeHookErr,
		},
		{
			name:          "returns repository errors",
			accountID:     "acc-1",
			providerID:    "google",
			createErr:     createErr,
			wantAccountID: "acc-1",
			wantErr:       createErr,
			expectCreate:  true,
		},
		{
			name:          "logs after create hook errors without failing",
			accountID:     "acc-1",
			providerID:    "google",
			afterHook:     func(a *models.Account) error { return afterHookErr },
			wantAccountID: "acc-1",
			wantLogs:      1,
			expectCreate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(internaltests.RecordingLogger)
			hooks := newAccountHooks()
			if tt.beforeHook != nil {
				hooks.Accounts.RegisterBeforeCreate(tt.beforeHook)
			}
			if tt.afterHook != nil {
				hooks.Accounts.RegisterAfterCreate(tt.afterHook)
			}
			svc, repo, tokenRepo := newAccountServiceWithHooksForTest(hooks, logger)

			created := &models.Account{ID: "created-account"}
			var captured *models.Account
			if tt.expectCreate {
				matcher := mock.MatchedBy(func(a *models.Account) bool {
					captured = a
					return true
				})
				if tt.createErr != nil {
					repo.On("Create", mock.Anything, matcher).Return(nil, tt.createErr)
				} else {
					repo.On("Create", mock.Anything, matcher).Return(created, nil)
				}
			}

			got, err := svc.Create(context.Background(), "user-1", tt.accountID, tt.providerID, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Same(t, created, got)
			}

			if tt.expectCreate {
				require.NotNil(t, captured)
				assert.NotEmpty(t, captured.ID)
				assert.Equal(t, "user-1", captured.UserID)
				assert.Equal(t, tt.wantAccountID, captured.AccountID)
				assert.Equal(t, tt.providerID, captured.ProviderID)
				assert.Equal(t, tt.password, captured.Password)
			} else {
				repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}

			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_CreateOAuth2_ValidationAndEncryptionErrors(t *testing.T) {
	t.Parallel()

	expired := time.Now().Add(-time.Hour)
	accessEncryptErr := errors.New("access token encryption failed")
	refreshEncryptErr := errors.New("refresh token encryption failed")

	tests := []struct {
		name                string
		accessExpiry        *time.Time
		accessEncryptErr    error
		refreshToken        *string
		refreshEncryptErr   error
		wantErr             error
		expectAccessEncrypt bool
	}{
		{
			name:         "rejects an expired access token",
			accessExpiry: &expired,
			wantErr:      coreservices.ErrTokenExpired,
		},
		{
			name:                "returns access token encryption errors",
			accessEncryptErr:    accessEncryptErr,
			wantErr:             accessEncryptErr,
			expectAccessEncrypt: true,
		},
		{
			name:                "returns refresh token encryption errors",
			refreshToken:        new("refresh"),
			refreshEncryptErr:   refreshEncryptErr,
			wantErr:             refreshEncryptErr,
			expectAccessEncrypt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenRepo := newAccountServiceForTest()
			if tt.expectAccessEncrypt {
				tokenRepo.On("Encrypt", "access").Return("", tt.accessEncryptErr)
			}
			if tt.refreshEncryptErr != nil {
				tokenRepo.On("Encrypt", "refresh").Return("", tt.refreshEncryptErr)
			}

			got, err := svc.CreateOAuth2(
				context.Background(),
				"user-1",
				"google-1",
				"google",
				"access",
				tt.refreshToken,
				tt.accessExpiry,
				nil,
				nil,
			)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_CreateOAuth2_CreatePath(t *testing.T) {
	t.Parallel()

	accessExpiry := time.Now().Add(time.Hour)
	refreshExpiry := time.Now().Add(24 * time.Hour)
	scope := "read write"
	lookupErr := errors.New("lookup failed")
	createErr := errors.New("create failed")
	beforeHookErr := errors.New("before create hook failed")
	afterHookErr := errors.New("after create hook failed")

	tests := []struct {
		name         string
		refreshToken *string
		lookupErr    error
		createErr    error
		beforeHook   models.ServiceHook[models.Account]
		afterHook    models.ServiceHook[models.Account]
		wantRefresh  *string
		wantErr      error
		wantLogs     int
		expectCreate bool
	}{
		{
			name:         "creates an account with encrypted access and refresh tokens",
			refreshToken: new("refresh"),
			wantRefresh:  new("encrypted-refresh"),
			expectCreate: true,
		},
		{
			name:         "creates an account without a refresh token",
			expectCreate: true,
		},
		{
			name:      "returns lookup errors",
			lookupErr: lookupErr,
			wantErr:   lookupErr,
		},
		{
			name:         "returns repository create errors",
			createErr:    createErr,
			wantErr:      createErr,
			expectCreate: true,
		},
		{
			name:       "returns before create hook errors",
			beforeHook: func(a *models.Account) error { return beforeHookErr },
			wantErr:    beforeHookErr,
		},
		{
			name:         "logs after create hook errors without failing",
			afterHook:    func(a *models.Account) error { return afterHookErr },
			wantLogs:     1,
			expectCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(internaltests.RecordingLogger)
			hooks := newAccountHooks()
			if tt.beforeHook != nil {
				hooks.Accounts.RegisterBeforeCreate(tt.beforeHook)
			}
			if tt.afterHook != nil {
				hooks.Accounts.RegisterAfterCreate(tt.afterHook)
			}
			svc, repo, tokenRepo := newAccountServiceWithHooksForTest(hooks, logger)

			tokenRepo.On("Encrypt", "access").Return("encrypted-access", nil)
			if tt.refreshToken != nil {
				tokenRepo.On("Encrypt", "refresh").Return("encrypted-refresh", nil)
			}
			repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(nil, tt.lookupErr)

			created := &models.Account{ID: "created-account"}
			var captured *models.Account
			if tt.expectCreate {
				matcher := mock.MatchedBy(func(a *models.Account) bool {
					captured = a
					return true
				})
				if tt.createErr != nil {
					repo.On("Create", mock.Anything, matcher).Return(nil, tt.createErr)
				} else {
					repo.On("Create", mock.Anything, matcher).Return(created, nil)
				}
			}

			got, err := svc.CreateOAuth2(
				context.Background(),
				"user-1",
				"google-1",
				"google",
				"access",
				tt.refreshToken,
				&accessExpiry,
				&refreshExpiry,
				&scope,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Same(t, created, got)
			}

			if tt.expectCreate {
				require.NotNil(t, captured)
				assert.NotEmpty(t, captured.ID)
				assert.Equal(t, "user-1", captured.UserID)
				assert.Equal(t, "google-1", captured.AccountID)
				assert.Equal(t, "google", captured.ProviderID)
				assert.Equal(t, new("encrypted-access"), captured.AccessToken)
				assert.Equal(t, tt.wantRefresh, captured.RefreshToken)
				assert.Equal(t, &accessExpiry, captured.AccessTokenExpiresAt)
				assert.Equal(t, &refreshExpiry, captured.RefreshTokenExpiresAt)
				assert.Equal(t, &scope, captured.Scope)
				assert.Nil(t, captured.Password)
			} else {
				repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}

			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_CreateOAuth2_MergesExistingAccount(t *testing.T) {
	t.Parallel()

	accessExpiry := time.Now().Add(time.Hour)
	refreshExpiry := time.Now().Add(24 * time.Hour)
	scope := "read:user write:repo"
	existing := &models.Account{
		ID:           "account-1",
		UserID:       "user-1",
		ProviderID:   "github",
		AccountID:    "github-1",
		AccessToken:  new("old-access"),
		RefreshToken: new("old-refresh"),
		Scope:        new("read:user"),
	}

	svc, repo, tokenRepo := newAccountServiceForTest()
	tokenRepo.On("Encrypt", "access").Return("encrypted-access", nil)
	tokenRepo.On("Encrypt", "refresh").Return("encrypted-refresh", nil)
	repo.On("GetByProviderAndAccountID", mock.Anything, "github", "github-1").Return(existing, nil)

	var captured *models.Account
	repo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Account) bool {
		captured = a
		return true
	})).Return(existing, nil)

	updated, err := svc.CreateOAuth2(
		context.Background(),
		"user-1",
		"github-1",
		"github",
		"access",
		new("refresh"),
		&accessExpiry,
		&refreshExpiry,
		&scope,
	)

	require.NoError(t, err)
	assert.Same(t, existing, updated)
	require.NotNil(t, captured)
	assert.Equal(t, new("encrypted-access"), captured.AccessToken)
	assert.Equal(t, new("encrypted-refresh"), captured.RefreshToken)
	assert.Equal(t, &accessExpiry, captured.AccessTokenExpiresAt)
	assert.Equal(t, &refreshExpiry, captured.RefreshTokenExpiresAt)
	assert.Equal(t, &scope, captured.Scope)
	repo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}

func TestAccountService_Update(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("update failed")
	beforeHookErr := errors.New("before update hook failed")
	afterHookErr := errors.New("after update hook failed")
	googleAccount := func() *models.Account {
		return &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
	}

	tests := []struct {
		name          string
		account       *models.Account
		beforeHook    models.ServiceHook[models.Account]
		afterHook     models.ServiceHook[models.Account]
		repoErr       error
		wantAccountID string
		wantErr       error
		wantLogs      int
		expectUpdate  bool
	}{
		{
			name:          "returns the repository result",
			account:       googleAccount(),
			wantAccountID: "google-1",
			expectUpdate:  true,
		},
		{
			name: "normalizes email account IDs before updating",
			account: &models.Account{
				ID:         "account-1",
				UserID:     "user-1",
				ProviderID: models.AuthProviderEmail.String(),
				AccountID:  "  User@Example.COM  ",
			},
			wantAccountID: "user@example.com",
			expectUpdate:  true,
		},
		{
			name:         "passes a nil account through to the repository",
			account:      nil,
			expectUpdate: true,
		},
		{
			name:          "returns repository errors",
			account:       googleAccount(),
			repoErr:       repoErr,
			wantAccountID: "google-1",
			wantErr:       repoErr,
			expectUpdate:  true,
		},
		{
			name:       "returns before update hook errors",
			account:    googleAccount(),
			beforeHook: func(a *models.Account) error { return beforeHookErr },
			wantErr:    beforeHookErr,
		},
		{
			name:          "logs after update hook errors without failing",
			account:       googleAccount(),
			afterHook:     func(a *models.Account) error { return afterHookErr },
			wantAccountID: "google-1",
			wantLogs:      1,
			expectUpdate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(internaltests.RecordingLogger)
			hooks := newAccountHooks()
			if tt.beforeHook != nil {
				hooks.Accounts.RegisterBeforeUpdate(tt.beforeHook)
			}
			if tt.afterHook != nil {
				hooks.Accounts.RegisterAfterUpdate(tt.afterHook)
			}
			svc, repo, tokenRepo := newAccountServiceWithHooksForTest(hooks, logger)

			updated := &models.Account{ID: "updated-account"}
			var captured *models.Account
			if tt.expectUpdate {
				matcher := mock.MatchedBy(func(a *models.Account) bool {
					captured = a
					return true
				})
				if tt.account == nil {
					repo.On("Update", mock.Anything, matcher).Return(nil, tt.repoErr)
				} else {
					repo.On("Update", mock.Anything, matcher).Return(updated, tt.repoErr)
				}
			}

			got, err := svc.Update(context.Background(), tt.account)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				if tt.account == nil {
					assert.Nil(t, got)
				} else {
					assert.Same(t, updated, got)
				}
			}

			if tt.expectUpdate {
				if tt.account == nil {
					assert.Nil(t, captured)
				} else {
					require.NotNil(t, captured, "repository must receive the account")
					assert.Equal(t, tt.wantAccountID, captured.AccountID)
				}
			} else {
				repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			}

			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}

func TestAccountService_UpdateFields(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("update fields failed")
	fields := map[string]any{"provider_id": "google", "scope": "read"}

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "delegates fields to the repository"},
		{name: "returns repository errors", repoErr: repoErr, wantErr: repoErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenRepo := newAccountServiceForTest()
			repo.On("UpdateFields", mock.Anything, "user-1", fields).Return(tt.repoErr)

			err := svc.UpdateFields(context.Background(), "user-1", fields)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}
