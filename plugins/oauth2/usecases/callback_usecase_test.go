package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestCallbackUseCase_ResolveUser(t *testing.T) {
	t.Parallel()

	existingUser := &models.User{ID: "user-1", Email: "alice@example.com"}
	newUser := &models.User{ID: "user-2", Email: "bob@example.com"}

	tests := []struct {
		name      string
		userInfo  types.UserInfo
		setup     func(*internaltests.MockUserService, *internaltests.MockAccountService)
		wantUser  *models.User
		expectErr bool
	}{
		{
			name:      "rejects missing account id",
			userInfo:  types.UserInfo{Email: "alice@example.com", EmailVerified: true},
			setup:     func(*internaltests.MockUserService, *internaltests.MockAccountService) {},
			expectErr: true,
		},
		{
			name:      "rejects missing email",
			userInfo:  types.UserInfo{ProviderAccountID: "oid-1"},
			setup:     func(*internaltests.MockUserService, *internaltests.MockAccountService) {},
			expectErr: true,
		},
		{
			name:     "known provider account returns its user even when email changed",
			userInfo: types.UserInfo{ProviderAccountID: "oid-1", Email: "renamed@example.com"},
			setup: func(users *internaltests.MockUserService, accounts *internaltests.MockAccountService) {
				accounts.On("GetByProviderAndAccountID", mock.Anything, "microsoft", "oid-1").
					Return(&models.Account{UserID: "user-1"}, nil)
				users.On("GetByID", mock.Anything, "user-1").Return(existingUser, nil)
			},
			wantUser: existingUser,
		},
		{
			name:     "verified email links to existing user",
			userInfo: types.UserInfo{ProviderAccountID: "g-1", Email: "alice@example.com", EmailVerified: true},
			setup: func(users *internaltests.MockUserService, accounts *internaltests.MockAccountService) {
				accounts.On("GetByProviderAndAccountID", mock.Anything, "microsoft", "g-1").Return(nil, nil)
				users.On("GetByEmail", mock.Anything, "alice@example.com").Return(existingUser, nil)
			},
			wantUser: existingUser,
		},
		{
			name:     "unverified email never links to existing user",
			userInfo: types.UserInfo{ProviderAccountID: "oid-attacker", Email: "alice@example.com"},
			setup: func(users *internaltests.MockUserService, accounts *internaltests.MockAccountService) {
				accounts.On("GetByProviderAndAccountID", mock.Anything, "microsoft", "oid-attacker").Return(nil, nil)
				users.On("GetByEmail", mock.Anything, "alice@example.com").Return(existingUser, nil)
			},
			expectErr: true,
		},
		{
			name:     "new user keeps provider email verification status",
			userInfo: types.UserInfo{ProviderAccountID: "oid-2", Email: "bob@example.com", Name: "Bob"},
			setup: func(users *internaltests.MockUserService, accounts *internaltests.MockAccountService) {
				accounts.On("GetByProviderAndAccountID", mock.Anything, "microsoft", "oid-2").Return(nil, nil)
				users.On("GetByEmail", mock.Anything, "bob@example.com").Return(nil, nil)
				users.On("Create", mock.Anything, "Bob", "bob@example.com", false, mock.Anything, mock.Anything).
					Return(newUser, nil)
			},
			wantUser: newUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			users := &internaltests.MockUserService{}
			accounts := &internaltests.MockAccountService{}
			tt.setup(users, accounts)

			uc := &CallbackUseCase{
				Logger:         &internaltests.MockLogger{},
				UserService:    users,
				AccountService: accounts,
			}

			user, err := uc.resolveUser(context.Background(), "microsoft", &tt.userInfo)
			if tt.expectErr {
				require.Error(t, err)
				users.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantUser, user)
			users.AssertExpectations(t)
			accounts.AssertExpectations(t)
		})
	}
}
