package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/services"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestChangePasswordUseCaseHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name: "before change password hook fires with new password",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					PasswordChange: &types.PasswordChangeServiceHooksConfig{
						BeforeChangePassword: func(ctx context.Context, user *models.User, newPassword string) error {
							require.Equal(t, "newpassword123", newPassword)
							require.Equal(t, "user-1", user.ID)
							return nil
						},
					},
				}, f.logger, nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(&models.Verification{
					ID: "ver-1", UserID: new("user-1"), Type: models.TypePasswordResetRequest, ExpiresAt: time.Now().Add(time.Hour), Identifier: "test@example.com",
				}, nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("oldhash")}, nil)
				f.passwordSvc.On("Hash", mock.Anything).Return("newhash", nil)
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("newhash")}, nil)
				f.verificationSvc.On("Delete", mock.Anything, "ver-1").Return(nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
				f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "before change password hook rejects",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					PasswordChange: &types.PasswordChangeServiceHooksConfig{
						BeforeChangePassword: func(ctx context.Context, user *models.User, newPassword string) error {
							return errors.New("password in history")
						},
					},
				}, f.logger, nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(&models.Verification{
					ID: "ver-1", UserID: new("user-1"), Type: models.TypePasswordResetRequest, ExpiresAt: time.Now().Add(time.Hour), Identifier: "test@example.com",
				}, nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("oldhash")}, nil)
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "password in history", err.Error())
			},
		},
		{
			name: "after change password hook fires",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					PasswordChange: &types.PasswordChangeServiceHooksConfig{
						AfterChangePassword: func(ctx context.Context, user *models.User) error {
							require.Equal(t, "user-1", user.ID)
							return nil
						},
					},
				}, f.logger, nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(&models.Verification{
					ID: "ver-1", UserID: new("user-1"), Type: models.TypePasswordResetRequest, ExpiresAt: time.Now().Add(time.Hour), Identifier: "test@example.com",
				}, nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("oldhash")}, nil)
				f.passwordSvc.On("Hash", mock.Anything).Return("newhash", nil)
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "acc-1", Password: new("newhash")}, nil)
				f.verificationSvc.On("Delete", mock.Anything, "ver-1").Return(nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
				f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
			err := f.changePasswordUseCase().ChangePassword(testRequestContext(), "token-123", "newpassword123")
			tc.assert(t, err)
			f.tokenSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
		})
	}
}
