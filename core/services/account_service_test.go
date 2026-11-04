package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func newAccountServiceForTest() (interface {
	CreateOAuth2(ctx context.Context, userID string, providerAccountID string, provider string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error)
	UpdateOAuth2Tokens(ctx context.Context, accountID string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error)
}, *internaltests.MockAccountRepository, *internaltests.MockTokenRepository) {
	repo := &internaltests.MockAccountRepository{}
	tokenRepo := &internaltests.MockTokenRepository{}
	svc := coreservices.NewAccountService(nil, repo, tokenRepo, nil, &internaltests.MockLogger{})
	return svc, repo, tokenRepo
}

func TestAccountService_UpdateOAuth2Tokens(t *testing.T) {
	t.Parallel()

	now := time.Now()
	future := now.Add(time.Hour)
	storedRefresh := "encrypted-old-refresh"
	storedRefreshExpiry := now.Add(48 * time.Hour)
	storedScope := "old-scope"

	tests := []struct {
		name                string
		account             *models.Account
		accessToken         string
		refreshToken        *string
		accessExpiry        *time.Time
		refreshExpiry       *time.Time
		scope               *string
		encryptAccess       string
		encryptRefresh      *string
		wantRefresh         *string
		wantRefreshExpiry   *time.Time
		wantScope           *string
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
			name:          "rejects expired token",
			accessToken:   "new-access",
			accessExpiry:  new(now.Add(-time.Minute)),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenRepo := newAccountServiceForTest()
			if tt.expectGetByID {
				repo.On("GetByID", mock.Anything, "account-1").Return(tt.account, nil)
			}
			if tt.expectEncryptAccess {
				tokenRepo.On("Encrypt", tt.accessToken).Return(tt.encryptAccess, nil)
			}
			if tt.encryptRefresh != nil {
				tokenRepo.On("Encrypt", "new-refresh").Return(*tt.encryptRefresh, nil)
			}
			if tt.expectGetByID && tt.account != nil {
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

func TestAccountService_CreateOAuth2RejectsAnotherOwner(t *testing.T) {
	t.Parallel()

	svc, repo, tokenRepo := newAccountServiceForTest()
	existing := &models.Account{ID: "account-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}

	tokenRepo.On("Encrypt", "access").Return("encrypted-access", nil)
	repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(existing, nil)

	_, err := svc.CreateOAuth2(context.Background(), "user-2", "google-1", "google", "access", nil, nil, nil, nil)

	require.ErrorIs(t, err, coreservices.ErrAccountLinkedToAnotherUser)
	repo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}

func TestAccountService_CreateOAuth2PreservesStoredRefreshToken(t *testing.T) {
	t.Parallel()

	svc, repo, tokenRepo := newAccountServiceForTest()
	storedRefresh := "encrypted-old-refresh"
	existing := &models.Account{
		ID:           "account-1",
		UserID:       "user-1",
		ProviderID:   "google",
		AccountID:    "google-1",
		RefreshToken: &storedRefresh,
	}

	tokenRepo.On("Encrypt", "access").Return("encrypted-access", nil)
	repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(existing, nil)

	updated, err := svc.CreateOAuth2(context.Background(), "user-1", "google-1", "google", "access", new(""), nil, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, &storedRefresh, updated.RefreshToken)
	repo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}
