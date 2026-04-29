package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSignInUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *types.SignInResult, error)
	}{
		{
			name: "invalid credentials when user missing",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidCredentials)
				require.Nil(t, result)
			},
		},
		{
			name: "invalid credentials when password mismatches",
			setup: func(f *emailPasswordTestFixture) {
				password := "hashed"
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", Password: &password}, nil).Once()
				f.passwordSvc.On("Verify", "wrong-password", "hashed").Return(false).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidCredentials)
				require.Nil(t, result)
			},
		},
		{
			name: "signs in existing user",
			setup: func(f *emailPasswordTestFixture) {
				password := "hashed-password"
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", Password: &password}, nil).Once()
				f.passwordSvc.On("Verify", "Password1!", "hashed-password").Return(true).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session-token").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session-token", (*string)(nil), (*string)(nil), time.Hour).Return(&models.Session{ID: "session-1"}, nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "session-token", result.SessionToken)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			password := "Password1!"
			if tc.name == "invalid credentials when password mismatches" {
				password = "wrong-password"
			}
			result, err := f.signInUseCase().SignIn(context.Background(), "test@example.com", password, nil, nil, nil)
			tc.assert(t, result, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
