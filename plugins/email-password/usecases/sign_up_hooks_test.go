package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/services"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSignUpUseCaseHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setup        func(*emailPasswordTestFixture)
		hookCallback func(*testing.T)
		assert       func(*testing.T, *types.SignUpResult, error)
	}{
		{
			name: "before sign up hook fires with user",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignUp: &types.SignUpServiceHooksConfig{
						BeforeSignUp: func(ctx context.Context, user *models.User) error {
							require.Equal(t, "Test User", user.Name)
							require.Equal(t, "test@example.com", user.Email)
							return nil
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil)
				f.userSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.User{ID: "user-1", Name: "Test User", Email: "test@example.com"}, nil)
				f.passwordSvc.On("Hash", mock.Anything).Return("hashed", nil)
				f.accountSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Account{ID: "acc-1"}, nil)
				f.tokenSvc.On("Generate").Return("token-123", nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.sessionSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "session-1"}, nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
			},
		},
		{
			name: "before sign up hook error rejects sign up",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignUp: &types.SignUpServiceHooksConfig{
						BeforeSignUp: func(ctx context.Context, user *models.User) error {
							return errors.New("hook rejected")
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil)
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.Error(t, err)
				require.Equal(t, "hook rejected", err.Error())
				require.Nil(t, result)
			},
		},
		{
			name: "after sign up hook fires with result",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignUp: &types.SignUpServiceHooksConfig{
						AfterSignUp: func(ctx context.Context, result *types.SignUpResult) error {
							require.Equal(t, "user-1", result.User.ID)
							require.NotNil(t, result.Session)
							return nil
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil)
				f.userSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.User{ID: "user-1", Name: "Test User", Email: "test@example.com"}, nil)
				f.passwordSvc.On("Hash", mock.Anything).Return("hashed", nil)
				f.accountSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Account{ID: "acc-1"}, nil)
				f.tokenSvc.On("Generate").Return("token-123", nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.sessionSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "session-1"}, nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tc.setup(f)
			result, err := f.signUpUseCase().SignUp(testRequestContext(), "Test User", "test@example.com", "password123", nil, nil, nil, nil, nil)
			tc.assert(t, result, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
