package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
)

func TestChangePasswordUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name:   "invalid length returns error",
			setup:  func(f *emailPasswordTestFixture) {},
			assert: func(t *testing.T, err error) { require.ErrorIs(t, err, constants.ErrInvalidPasswordLength) },
		},
		{
			name: "updates password on valid token",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypePasswordResetRequest, UserID: func() *string { s := "user-1"; return &s }(), ExpiresAt: time.Now().Add(time.Hour)}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				f.passwordSvc.On("Hash", "NewPassword1!").Return("hashed-password", nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Account) bool { return a.Password != nil && *a.Password == "hashed-password" })).Return(&models.Account{ID: "account-1"}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			password := "short"
			if tc.name == "updates password on valid token" {
				password = "NewPassword1!"
			}
			err := f.changePasswordUseCase().ChangePassword(context.Background(), "token-123", password)
			tc.assert(t, err)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
		})
	}
}
