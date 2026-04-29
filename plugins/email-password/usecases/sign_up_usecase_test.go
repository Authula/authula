package usecases

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSignUpUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *types.SignUpResult, error)
	}{
		{
			name:  "disabled sign up returns error",
			setup: func(f *emailPasswordTestFixture) { f.pluginConfig.DisableSignUp = true },
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.ErrorIs(t, err, constants.ErrSignUpDisabled)
				require.Nil(t, result)
			},
		},
		{
			name:  "invalid password length returns error",
			setup: func(f *emailPasswordTestFixture) { f.pluginConfig.MinPasswordLength = 12 },
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidPasswordLength)
				require.Nil(t, result)
			},
		},
		{
			name: "existing email returns conflict",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.ErrorIs(t, err, constants.ErrEmailAlreadyExists)
				require.Nil(t, result)
			},
		},
		{
			name: "creates user account and session",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
				f.passwordSvc.On("Hash", "Password1!").Return("hashed-password", nil).Once()
				f.userSvc.On("Create", mock.Anything, "Test User", "test@example.com", false, (*string)(nil), json.RawMessage(`{"role":"member"}`)).Return(&models.User{ID: "user-1", Name: "Test User", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("Create", mock.Anything, "user-1", "test@example.com", models.AuthProviderEmail.String(), mock.AnythingOfType("*string")).Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session-token").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session-token", (*string)(nil), (*string)(nil), time.Hour).Return(&models.Session{ID: "session-1", UserID: "user-1"}, nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "session-token", result.SessionToken)
				require.Equal(t, "user-1", result.User.ID)
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
			result, err := f.signUpUseCase().SignUp(context.Background(), "Test User", "test@example.com", "Password1!", nil, json.RawMessage(`{"role":"member"}`), nil, nil, nil)
			tc.assert(t, result, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
