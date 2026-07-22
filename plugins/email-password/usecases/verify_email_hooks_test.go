package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/services"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestVerifyEmailUseCaseHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, models.VerificationType, error)
	}{
		{
			name: "after verify email hook fires for email verification type",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					EmailVerification: &types.EmailVerificationServiceHooksConfig{
						AfterVerifyEmail: func(ctx context.Context, user *models.User, verificationType models.VerificationType) error {
							require.Equal(t, "user-1", user.ID)
							require.Equal(t, models.TypeEmailVerification, verificationType)
							return nil
						},
					},
				}, f.logger)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(&models.Verification{
					ID: "ver-1", UserID: new("user-1"), Type: models.TypeEmailVerification, ExpiresAt: time.Now().Add(time.Hour), Identifier: "test@example.com",
				}, nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com", EmailVerified: false}, nil)
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: "user-1", Email: "test@example.com", EmailVerified: true}, nil)
				f.verificationSvc.On("Delete", mock.Anything, "ver-1").Return(nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, verType models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailVerification, verType)
			},
		},
		{
			name: "after verify email hook fires for password reset type",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					EmailVerification: &types.EmailVerificationServiceHooksConfig{
						AfterVerifyEmail: func(ctx context.Context, user *models.User, verificationType models.VerificationType) error {
							require.Equal(t, models.TypePasswordResetRequest, verificationType)
							return nil
						},
					},
				}, f.logger)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(&models.Verification{
					ID: "ver-1", UserID: new("user-1"), Type: models.TypePasswordResetRequest, ExpiresAt: time.Now().Add(time.Hour), Identifier: "test@example.com",
				}, nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil)
			},
			assert: func(t *testing.T, verType models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypePasswordResetRequest, verType)
			},
		},
		{
			name: "after verify email hook fires for email reset type",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					EmailVerification: &types.EmailVerificationServiceHooksConfig{
						AfterVerifyEmail: func(ctx context.Context, user *models.User, verificationType models.VerificationType) error {
							require.Equal(t, models.TypeEmailResetRequest, verificationType)
							return nil
						},
					},
				}, f.logger)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(&models.Verification{
					ID: "ver-1", UserID: new("user-1"), Type: models.TypeEmailResetRequest, ExpiresAt: time.Now().Add(time.Hour), Identifier: "new@example.com",
				}, nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "old@example.com"}, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", mock.Anything).Return(&models.Account{ID: "acc-1", AccountID: "old@example.com"}, nil)
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: "user-1", Email: "new@example.com"}, nil)
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "acc-1", AccountID: "new@example.com"}, nil)
				f.verificationSvc.On("Delete", mock.Anything, "ver-1").Return(nil)
				f.eventBus.On("Publish", mock.Anything).Return(nil)
				f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, verType models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailResetRequest, verType)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tc.setup(f)
			verType, err := f.verifyEmailUseCase().VerifyEmail(testRequestContext(), "token-123")
			tc.assert(t, verType, err)
			f.tokenSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.userSvc.AssertExpectations(t)
		})
	}
}
