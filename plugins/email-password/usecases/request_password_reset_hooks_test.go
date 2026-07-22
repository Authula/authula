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

func TestRequestPasswordResetUseCaseHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		email  string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name:  "before request password reset hook fires with user",
			email: "test@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					PasswordReset: &types.PasswordResetServiceHooksConfig{
						BeforeRequestPasswordReset: func(ctx context.Context, user *models.User) error {
							require.Equal(t, "user-1", user.ID)
							return nil
						},
					},
				}, f.logger)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
				f.tokenSvc.On("Generate").Return("token-123", nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Verification{ID: "ver-1"}, nil)
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "before request password reset hook rejects",
			email: "test@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					PasswordReset: &types.PasswordResetServiceHooksConfig{
						BeforeRequestPasswordReset: func(ctx context.Context, user *models.User) error {
							return errors.New("rate limited")
						},
					},
				}, f.logger)
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "rate limited", err.Error())
			},
		},
		{
			name:  "before request password reset hook skipped for nil user",
			email: "nonexistent@example.com",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					PasswordReset: &types.PasswordResetServiceHooksConfig{
						BeforeRequestPasswordReset: func(ctx context.Context, user *models.User) error {
							t.Error("hook should not be called for nil user")
							return nil
						},
					},
				}, f.logger)
				f.userSvc.On("GetByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil)
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tc.setup(f)
			err := f.requestPasswordResetUseCase().RequestReset(testRequestContext(), tc.email, nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
		})
	}
}
