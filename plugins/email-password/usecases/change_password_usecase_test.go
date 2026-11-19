package usecases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	emailtmpl "github.com/Authula/authula/core/email/template"
	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestChangePasswordUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		setup    func(*emailPasswordTestFixture)
		assert   func(*testing.T, error)
	}{
		{
			name:     "invalid length returns error",
			password: "short",
			setup:    func(f *emailPasswordTestFixture) {},
			assert:   func(t *testing.T, err error) { require.ErrorIs(t, err, constants.ErrInvalidPasswordLength) },
		},
		{
			name:     "verification repository reports not found",
			password: "NewPassword1!",
			setup: func(f *emailPasswordTestFixture) {
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(nil, coreerrors.ErrVerificationNotFound).Once()
			},
			assert: func(t *testing.T, err error) { require.ErrorIs(t, err, constants.ErrInvalidOrExpiredToken) },
		},
		{
			name:     "updates password on valid token",
			password: "NewPassword1!",
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
				f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
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
			password := tc.password
			err := f.changePasswordUseCase().ChangePassword(context.Background(), "token-123", password)
			tc.assert(t, err)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
		})
	}
}

func passwordFlowVerification() *models.Verification {
	return &models.Verification{
		ID:        "verif-1",
		Type:      models.TypePasswordResetRequest,
		UserID:    new("user-1"),
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func expectPasswordFlowToken(f *emailPasswordTestFixture, verification *models.Verification, err error) {
	f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
	f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, err).Once()
}

// expectPasswordFlowOwnerAndAccount prepares the user and account lookups that
// precede the password update.
func expectPasswordFlowOwnerAndAccount(f *emailPasswordTestFixture) {
	f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
	f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", AccountID: "test@example.com"}, nil).Once()
}

func TestChangePasswordUseCase_ErrorPaths(t *testing.T) {
	t.Parallel()

	expiredVerification := func() *models.Verification {
		verification := passwordFlowVerification()
		verification.ExpiresAt = time.Now().Add(-time.Hour)
		return verification
	}
	wrongTypeVerification := func() *models.Verification {
		verification := passwordFlowVerification()
		verification.Type = models.TypeEmailResetRequest
		return verification
	}

	tests := []struct {
		name      string
		setup     func(*emailPasswordTestFixture)
		expectErr error
	}{
		{
			name: "unexpected token lookup error is returned",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, nil, errPasswordFlowBackend)
			},
			expectErr: errPasswordFlowBackend,
		},
		{
			name: "missing verification is rejected",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, nil, nil)
			},
			expectErr: constants.ErrInvalidOrExpiredToken,
		},
		{
			name: "expired verification is rejected",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, expiredVerification(), nil)
			},
			expectErr: constants.ErrInvalidOrExpiredToken,
		},
		{
			name: "wrong verification type is rejected",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, wrongTypeVerification(), nil)
			},
			expectErr: constants.ErrInvalidOrExpiredToken,
		},
		{
			name: "user lookup error is returned",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, errPasswordFlowBackend).Once()
			},
			expectErr: errPasswordFlowBackend,
		},
		{
			name: "missing user is rejected",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			expectErr: constants.ErrUserNotFound,
		},
		{
			name: "account lookup error is returned",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(nil, errPasswordFlowBackend).Once()
			},
			expectErr: errPasswordFlowBackend,
		},
		{
			name: "missing account is rejected",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(nil, nil).Once()
			},
			expectErr: constants.ErrAccountNotFound,
		},
		{
			name: "password hash error is returned",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				expectPasswordFlowOwnerAndAccount(f)
				f.passwordSvc.On("Hash", "NewPassword1!").Return("", errPasswordFlowBackend).Once()
			},
			expectErr: errPasswordFlowBackend,
		},
		{
			name: "account update error is returned",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				expectPasswordFlowOwnerAndAccount(f)
				f.passwordSvc.On("Hash", "NewPassword1!").Return("hashed-password", nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(nil, errPasswordFlowBackend).Once()
			},
			expectErr: errPasswordFlowBackend,
		},
		{
			name: "verification delete error is returned",
			setup: func(f *emailPasswordTestFixture) {
				expectPasswordFlowToken(f, passwordFlowVerification(), nil)
				expectPasswordFlowOwnerAndAccount(f)
				f.passwordSvc.On("Hash", "NewPassword1!").Return("hashed-password", nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "account-1"}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(errPasswordFlowBackend).Once()
			},
			expectErr: errPasswordFlowBackend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tt.setup(f)

			err := f.changePasswordUseCase().ChangePassword(context.Background(), "token-123", "NewPassword1!")

			require.ErrorIs(t, err, tt.expectErr)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
		})
	}
}

func TestChangePasswordUseCase_CallbackPaths(t *testing.T) {
	t.Parallel()

	expectCommit := func(f *emailPasswordTestFixture) {
		expectPasswordFlowToken(f, passwordFlowVerification(), nil)
		expectPasswordFlowOwnerAndAccount(f)
		f.passwordSvc.On("Hash", "NewPassword1!").Return("hashed-password", nil).Once()
		f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "account-1"}, nil).Once()
		f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
	}

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture, *passwordFlowLogger, chan struct{})
		assert func(*testing.T, *emailPasswordTestFixture, *passwordFlowLogger, <-chan struct{})
	}{
		{
			name: "successful callback skips the mailer",
			setup: func(f *emailPasswordTestFixture, _ *passwordFlowLogger, _ chan struct{}) {
				expectCommit(f)
				f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
				f.pluginConfig.SendChangedPasswordEmail = func(types.SendChangedPasswordEmailParams, *models.RequestContext) error {
					return nil
				}
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, _ *passwordFlowLogger, _ <-chan struct{}) {
				f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
			},
		},
		{
			name: "callback failure is logged and falls back to the mailer",
			setup: func(f *emailPasswordTestFixture, _ *passwordFlowLogger, sent chan struct{}) {
				expectCommit(f)
				f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
				f.pluginConfig.SendChangedPasswordEmail = func(types.SendChangedPasswordEmailParams, *models.RequestContext) error {
					return errPasswordFlowBackend
				}
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", "Your password has been changed", mock.Anything, mock.Anything).
					Run(func(mock.Arguments) { close(sent) }).
					Return(nil).Once()
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, logger *passwordFlowLogger, sent <-chan struct{}) {
				waitForPasswordFlowSignal(t, sent, "changed password email")
				require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send changed password email via plugin callback")
				f.mailerSvc.AssertExpectations(t)
			},
		},
		{
			name: "background send failure is logged",
			setup: func(f *emailPasswordTestFixture, _ *passwordFlowLogger, _ chan struct{}) {
				expectCommit(f)
				f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
				f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errPasswordFlowBackend).Once()
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, logger *passwordFlowLogger, _ <-chan struct{}) {
				waitForPasswordFlowSignal(t, logger.done, "background send error")
				require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send changed password email via built-in email service")
				f.mailerSvc.AssertExpectations(t)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newEmailPasswordTestFixture()
			logger := newPasswordFlowLogger()
			sent := make(chan struct{}, 1)
			tc.setup(f, logger, sent)

			uc := NewChangePasswordUseCase(f.globalConfig, logger, f.pluginConfig, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.passwordSvc, f.mailerSvc, f.eventBus, f.tmplMgr, f.hooksExecutor)
			err := uc.ChangePassword(context.Background(), "token-123", "NewPassword1!")

			require.NoError(t, err)
			tc.assert(t, f, logger, sent)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
		})
	}
}

func TestChangePasswordUseCase_SendChangedPasswordEmail(t *testing.T) {
	t.Parallel()

	const (
		userEmail = "test@example.com"
		subject   = "Your password has been changed"
	)

	tests := []struct {
		name    string
		setup   func(*emailPasswordTestFixture)
		wantErr string
		wantLog string
	}{
		{
			name: "renders the template and sends it",
			setup: func(f *emailPasswordTestFixture) {
				f.mailerSvc.On("SendEmail", mock.Anything, userEmail, subject, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "render failure is logged and returned",
			setup: func(f *emailPasswordTestFixture) {
				// A manager without registered templates forces the render path to fail.
				f.tmplMgr = emailtmpl.NewManager()
			},
			wantErr: "not found",
			wantLog: "failed to render password changed template",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newEmailPasswordTestFixture()
			logger := newPasswordFlowLogger()
			tc.setup(f)

			uc := NewChangePasswordUseCase(f.globalConfig, logger, f.pluginConfig, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.passwordSvc, f.mailerSvc, f.eventBus, f.tmplMgr, f.hooksExecutor).(*changePasswordUseCase)
			err := uc.sendChangedPasswordEmail(context.Background(), &models.User{ID: "user-1", Email: userEmail})

			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			if tc.wantLog != "" {
				require.Contains(t, strings.Join(logger.messages(), "\n"), tc.wantLog)
			}
			f.mailerSvc.AssertExpectations(t)
			if tc.wantErr != "" {
				f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
			}
		})
	}
}
