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

func TestSignInUseCaseHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		email  string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *types.SignInResult, error)
	}{
		{
			name:  "before sign in hook fires with user",
			email: "test@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignIn: &types.SignInServiceHooksConfig{
						BeforeSignIn: func(ctx context.Context, user *models.User) error {
							require.Equal(t, "user-1", user.ID)
							return nil
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("hashed")}, nil)
				f.passwordSvc.On("Verify", mock.Anything, mock.Anything).Return(true)
				f.tokenSvc.On("Generate").Return("token-123", nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.sessionSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "session-1"}, nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
			},
		},
		{
			name:  "before sign in hook rejects sign in",
			email: "test@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignIn: &types.SignInServiceHooksConfig{
						BeforeSignIn: func(ctx context.Context, user *models.User) error {
							return errors.New("hook rejected")
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.Error(t, err)
				require.Equal(t, "hook rejected", err.Error())
				require.Nil(t, result)
			},
		},
		{
			name:  "before sign in hook skipped for nil user",
			email: "nonexistent@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignIn: &types.SignInServiceHooksConfig{
						BeforeSignIn: func(ctx context.Context, user *models.User) error {
							t.Error("hook should not be called for nil user")
							return nil
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil)
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.Error(t, err)
				require.Nil(t, result)
			},
		},
		{
			name:  "after sign in hook fires with result",
			email: "test@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					SignIn: &types.SignInServiceHooksConfig{
						AfterSignIn: func(ctx context.Context, result *types.SignInResult) error {
							require.Equal(t, "session-1", result.Session.ID)
							require.Equal(t, "token-123", result.SessionToken)
							return nil
						},
					},
				}, f.logger, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("hashed")}, nil)
				f.passwordSvc.On("Verify", mock.Anything, mock.Anything).Return(true)
				f.tokenSvc.On("Generate").Return("token-123", nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.sessionSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "session-1"}, nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
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
			result, err := f.signInUseCase().SignIn(testRequestContext(), tc.email, "password123", nil, nil, nil)
			tc.assert(t, result, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
