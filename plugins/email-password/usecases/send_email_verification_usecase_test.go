package usecases

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSendEmailVerificationUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name:   "disabled verification returns nil",
			setup:  func(f *emailPasswordTestFixture) { f.pluginConfig.RequireEmailVerification = false },
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "missing user is swallowed",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "verified user returns nil",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com", EmailVerified: true}, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "creates verification and calls hook",
			setup: func(f *emailPasswordTestFixture) {
				f.pluginConfig.SendEmailVerification = func(params types.SendEmailVerificationParams, reqCtx *models.RequestContext) error {
					require.Equal(t, "test@example.com", params.User.Email)
					require.NotNil(t, reqCtx)
					return nil
				}
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("verify-token", nil).Once()
				f.tokenSvc.On("Hash", "verify-token").Return("hashed-token").Once()
				f.verificationSvc.On("DeleteByUserIDAndType", mock.Anything, "user-1", models.TypeEmailVerification).Return(nil).Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailVerification, "test@example.com", 24*time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "current-user email drives lookup and verification",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "current@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("verify-token", nil).Once()
				f.tokenSvc.On("Hash", "verify-token").Return("hashed-token").Once()
				f.verificationSvc.On("DeleteByUserIDAndType", mock.Anything, "user-1", models.TypeEmailVerification).Return(nil).Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailVerification, "current@example.com", 24*time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "current@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
			err := f.sendEmailVerificationUseCase().Send(testRequestContext(), "user-1", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
