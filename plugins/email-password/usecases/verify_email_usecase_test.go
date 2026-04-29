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

func TestVerifyEmailUseCase(t *testing.T) {
	t.Parallel()

	userID := "user-1"

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, models.VerificationType, error)
	}{
		{
			name: "invalid token returns error",
			setup: func(f *emailPasswordTestFixture) {
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidOrExpiredToken)
				require.Empty(t, vt)
			},
		},
		{
			name: "email verification marks user verified",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypeEmailVerification, UserID: &userID, ExpiresAt: time.Now().Add(time.Hour)}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "test@example.com"}, nil).Once()
				f.userSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.EmailVerified })).Return(&models.User{ID: userID, EmailVerified: true}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailVerification, vt)
			},
		},
		{
			name: "email reset request returns type",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypeEmailResetRequest, UserID: &userID, ExpiresAt: time.Now().Add(time.Hour), Identifier: "new@example.com"}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, userID, models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: userID}, nil).Once()
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: userID, Email: "new@example.com"}, nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "account-1"}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "old@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				f.mailerSvc.On("SendEmail", mock.Anything, "new@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailResetRequest, vt)
			},
		},
		{
			name: "email reset callback returns type",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypeEmailResetRequest, UserID: &userID, ExpiresAt: time.Now().Add(time.Hour), Identifier: "new@example.com"}
				f.pluginConfig.SendChangedEmailToOldEmail = func(params types.SendChangedEmailToOldEmailParams, reqCtx *models.RequestContext) error {
					require.Equal(t, "old@example.com", params.Email)
					require.NotNil(t, reqCtx)
					return nil
				}
				f.pluginConfig.SendChangedEmailToNewEmail = func(params types.SendChangedEmailToNewEmailParams, reqCtx *models.RequestContext) error {
					require.Equal(t, "new@example.com", params.Email)
					require.NotNil(t, reqCtx)
					return nil
				}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, userID, models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: userID}, nil).Once()
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: userID, Email: "new@example.com"}, nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "account-1"}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailResetRequest, vt)
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
			ctx := models.SetRequestContext(context.Background(), &models.RequestContext{Values: map[string]any{}})
			vt, err := f.verifyEmailUseCase().VerifyEmail(ctx, "token-123")
			tc.assert(t, vt, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
