package usecases

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	emailtmpl "github.com/Authula/authula/core/email/template"
	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/services"
	"github.com/Authula/authula/plugins/email-password/types"
)

const (
	emailChangeUserID   = "user-1"
	emailChangeOldEmail = "old@example.com"
	emailChangeNewEmail = "new@example.com"
)

var errEmailChangeBackend = errors.New("database unavailable")

// emailChangeNotifyingLogger records error messages and signals the first one,
// so tests can wait for logging that happens in a background goroutine.
type emailChangeNotifyingLogger struct {
	mu     sync.Mutex
	errors []string
	done   chan struct{}
	once   sync.Once
}

func newEmailChangeNotifyingLogger() *emailChangeNotifyingLogger {
	return &emailChangeNotifyingLogger{done: make(chan struct{})}
}

func (l *emailChangeNotifyingLogger) Debug(string, ...any) {}
func (l *emailChangeNotifyingLogger) Info(string, ...any)  {}
func (l *emailChangeNotifyingLogger) Warn(string, ...any)  {}

func (l *emailChangeNotifyingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.errors = append(l.errors, msg)
	l.mu.Unlock()
	l.once.Do(func() { close(l.done) })
}

func (l *emailChangeNotifyingLogger) messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.errors...)
}

func waitForEmailChangeSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

// newEmailChangeFixture adds the notification template that the production
// template definitions register under "email_changed". The shared fixture only
// registers "email_changed_notification", which the renderer never looks up.
func newEmailChangeFixture(t *testing.T) *emailPasswordTestFixture {
	t.Helper()
	f := newEmailPasswordTestFixture()
	require.NoError(t, f.tmplMgr.Register(emailtmpl.Definition{
		Name:    "email_changed",
		Subject: "Your email has been changed",
		Text:    "Changed from {{.OldEmail}} to {{.NewEmail}}",
		HTML:    "<p>Changed from {{.OldEmail}} to {{.NewEmail}}</p>",
	}))
	return f
}

func emailChangeVerification() *models.Verification {
	return &models.Verification{
		ID:         "ver-1",
		Type:       models.TypeEmailResetRequest,
		UserID:     new(emailChangeUserID),
		Identifier: emailChangeNewEmail,
		ExpiresAt:  time.Now().Add(time.Hour),
	}
}

func expectEmailChangeToken(f *emailPasswordTestFixture, verification *models.Verification, err error) {
	f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
	f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, err).Once()
}

func expectEmailChangeOwner(f *emailPasswordTestFixture, user *models.User, err error) {
	f.userSvc.On("GetByID", mock.Anything, emailChangeUserID).Return(user, err).Once()
}

func expectEmailChangeNoExistingEmail(f *emailPasswordTestFixture) {
	f.userSvc.On("GetByEmail", mock.Anything, emailChangeNewEmail).Return(nil, nil).Once()
}

func expectEmailChangeAccount(f *emailPasswordTestFixture, account *models.Account, err error) {
	f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, emailChangeUserID, models.AuthProviderEmail.String()).Return(account, err).Once()
}

func expectEmailChangeCommit(f *emailPasswordTestFixture) {
	f.userSvc.On("Update", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
		return user != nil && user.Email == emailChangeNewEmail && user.EmailVerified
	})).Return(&models.User{ID: emailChangeUserID, Email: emailChangeNewEmail, EmailVerified: true}, nil).Once()
	f.accountSvc.On("Update", mock.Anything, mock.MatchedBy(func(account *models.Account) bool {
		return account != nil && account.AccountID == emailChangeNewEmail
	})).Return(&models.Account{ID: "acc-1", AccountID: emailChangeNewEmail}, nil).Once()
	f.verificationSvc.On("Delete", mock.Anything, "ver-1").Return(nil).Once()
}

// expectEmailChangeHappyPath prepares every dependency up to the commit phase.
func expectEmailChangeHappyPath(f *emailPasswordTestFixture) {
	expectEmailChangeToken(f, emailChangeVerification(), nil)
	expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
	expectEmailChangeNoExistingEmail(f)
	expectEmailChangeAccount(f, &models.Account{ID: "acc-1", UserID: emailChangeUserID, AccountID: emailChangeOldEmail}, nil)
	expectEmailChangeCommit(f)
}

// expectEmailChangeEvent acknowledges the asynchronous telemetry publish.
func expectEmailChangeEvent(f *emailPasswordTestFixture) {
	f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
}

func TestVerifyEmailUseCase_HandleEmailChangeVerificationEmailErrors(t *testing.T) {
	t.Parallel()

	expiredVerification := func() *models.Verification {
		verification := emailChangeVerification()
		verification.ExpiresAt = time.Now().Add(-time.Hour)
		return verification
	}
	wrongTypeVerification := func() *models.Verification {
		verification := emailChangeVerification()
		verification.Type = models.TypeEmailVerification
		return verification
	}
	mismatchedVerification := func() *models.Verification {
		verification := emailChangeVerification()
		verification.Identifier = "other@example.com"
		return verification
	}

	tests := []struct {
		name           string
		newEmail       string
		setup          func(*emailPasswordTestFixture)
		expectErr      error
		expectContains string
	}{
		{
			name:           "empty new email is rejected",
			newEmail:       "",
			expectContains: "new email cannot be empty",
		},
		{
			name:     "unknown verification is mapped to invalid or expired token",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, nil, coreerrors.ErrVerificationNotFound)
			},
			expectErr: constants.ErrInvalidOrExpiredToken,
		},
		{
			name:     "token lookup error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, nil, errEmailChangeBackend)
			},
			expectErr: errEmailChangeBackend,
		},
		{
			name:     "expired verification is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, expiredVerification(), nil)
			},
			expectErr: constants.ErrInvalidOrExpiredToken,
		},
		{
			name:     "wrong verification type is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, wrongTypeVerification(), nil)
			},
			expectErr: constants.ErrInvalidOrExpiredToken,
		},
		{
			name:     "identifier mismatch is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, mismatchedVerification(), nil)
			},
			expectErr: constants.ErrInvalidEmailMatch,
		},
		{
			name:     "user lookup error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, nil, errEmailChangeBackend)
			},
			expectErr: errEmailChangeBackend,
		},
		{
			name:     "missing user is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, nil, nil)
			},
			expectErr: constants.ErrUserNotFound,
		},
		{
			name:     "different user is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: "user-2", Email: emailChangeOldEmail}, nil)
			},
			expectErr: constants.ErrUserNotAuthorized,
		},
		{
			name:     "email lookup error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				f.userSvc.On("GetByEmail", mock.Anything, emailChangeNewEmail).Return(nil, errEmailChangeBackend).Once()
			},
			expectErr: errEmailChangeBackend,
		},
		{
			name:     "email used by another user is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				f.userSvc.On("GetByEmail", mock.Anything, emailChangeNewEmail).Return(&models.User{ID: "user-2", Email: emailChangeNewEmail}, nil).Once()
			},
			expectErr: constants.ErrEmailAlreadyExists,
		},
		{
			name:     "account lookup error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				expectEmailChangeNoExistingEmail(f)
				expectEmailChangeAccount(f, nil, errEmailChangeBackend)
			},
			expectErr: errEmailChangeBackend,
		},
		{
			name:     "missing account is rejected",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				expectEmailChangeNoExistingEmail(f)
				expectEmailChangeAccount(f, nil, nil)
			},
			expectErr: constants.ErrAccountNotFound,
		},
		{
			name:     "user update error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				expectEmailChangeNoExistingEmail(f)
				expectEmailChangeAccount(f, &models.Account{ID: "acc-1", AccountID: emailChangeOldEmail}, nil)
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(nil, errEmailChangeBackend).Once()
			},
			expectErr: errEmailChangeBackend,
		},
		{
			name:     "account update error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				expectEmailChangeNoExistingEmail(f)
				expectEmailChangeAccount(f, &models.Account{ID: "acc-1", AccountID: emailChangeOldEmail}, nil)
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: emailChangeUserID, Email: emailChangeNewEmail}, nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(nil, errEmailChangeBackend).Once()
			},
			expectErr: errEmailChangeBackend,
		},
		{
			name:     "verification delete error is returned",
			newEmail: emailChangeNewEmail,
			setup: func(f *emailPasswordTestFixture) {
				expectEmailChangeToken(f, emailChangeVerification(), nil)
				expectEmailChangeOwner(f, &models.User{ID: emailChangeUserID, Email: emailChangeOldEmail}, nil)
				expectEmailChangeNoExistingEmail(f)
				expectEmailChangeAccount(f, &models.Account{ID: "acc-1", AccountID: emailChangeOldEmail}, nil)
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: emailChangeUserID, Email: emailChangeNewEmail}, nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "acc-1", AccountID: emailChangeNewEmail}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "ver-1").Return(errEmailChangeBackend).Once()
			},
			expectErr: errEmailChangeBackend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailChangeFixture(t)
			if tt.setup != nil {
				tt.setup(f)
			}

			uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
			err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", tt.newEmail)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.ErrorContains(t, err, tt.expectContains)
			}
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestVerifyEmailUseCase_HandleEmailChangeVerificationEmailSuccess(t *testing.T) {
	t.Parallel()

	t.Run("updates the user and account and deletes the verification", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)
		f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)

		require.NoError(t, err)
		f.userSvc.AssertExpectations(t)
		f.accountSvc.AssertExpectations(t)
		f.verificationSvc.AssertExpectations(t)
	})

	t.Run("after email changed hook receives the old and new email", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)
		f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
			EmailChange: &types.EmailChangeServiceHooksConfig{
				AfterEmailChanged: func(ctx context.Context, user *models.User, oldEmail, newEmail string) error {
					require.Equal(t, emailChangeOldEmail, oldEmail)
					require.Equal(t, emailChangeNewEmail, newEmail)
					require.Equal(t, emailChangeNewEmail, user.Email)
					return nil
				},
			},
		}, f.logger, nil, nil)
		f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)

		require.NoError(t, err)
		f.userSvc.AssertExpectations(t)
	})

	t.Run("mailer fallback sends both notifications", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)

		var mu sync.Mutex
		sent := 0
		done := make(chan struct{})
		record := func(mock.Arguments) {
			mu.Lock()
			defer mu.Unlock()
			sent++
			if sent == 2 {
				close(done)
			}
		}
		f.mailerSvc.On("SendEmail", mock.Anything, emailChangeOldEmail, "Your email has been changed", mock.Anything, mock.Anything).Run(record).Return(nil).Once()
		f.mailerSvc.On("SendEmail", mock.Anything, emailChangeNewEmail, "Your email has been changed", mock.Anything, mock.Anything).Run(record).Return(nil).Once()

		uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)
		require.NoError(t, err)

		waitForEmailChangeSignal(t, done, "both notification emails")
		f.mailerSvc.AssertExpectations(t)
	})

	t.Run("old email callback success skips the mailer", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)
		f.pluginConfig.SendChangedEmailToOldEmail = func(types.SendChangedEmailToOldEmailParams, *models.RequestContext) error {
			return nil
		}

		uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)

		require.NoError(t, err)
		f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
	})

	t.Run("new email callback success skips the mailer", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)
		f.pluginConfig.SendChangedEmailToNewEmail = func(types.SendChangedEmailToNewEmailParams, *models.RequestContext) error {
			return nil
		}

		uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)

		require.NoError(t, err)
		f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
	})

	t.Run("old email callback error is logged", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)
		logger := newEmailChangeNotifyingLogger()
		f.pluginConfig.SendChangedEmailToOldEmail = func(types.SendChangedEmailToOldEmailParams, *models.RequestContext) error {
			return errEmailChangeBackend
		}
		// A successful new-email callback marks the flow as handled, so no
		// background goroutine is started.
		f.pluginConfig.SendChangedEmailToNewEmail = func(types.SendChangedEmailToNewEmailParams, *models.RequestContext) error {
			return nil
		}

		uc := NewVerifyEmailUseCase(f.globalConfig, f.pluginConfig, logger, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.eventBus, f.tmplMgr).(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)

		require.NoError(t, err)
		require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send changed email to old email via plugin callback")
	})

	t.Run("new email callback error is logged", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		expectEmailChangeHappyPath(f)
		expectEmailChangeEvent(f)
		logger := newEmailChangeNotifyingLogger()
		f.pluginConfig.SendChangedEmailToOldEmail = func(types.SendChangedEmailToOldEmailParams, *models.RequestContext) error {
			return nil
		}
		f.pluginConfig.SendChangedEmailToNewEmail = func(types.SendChangedEmailToNewEmailParams, *models.RequestContext) error {
			return errEmailChangeBackend
		}

		uc := NewVerifyEmailUseCase(f.globalConfig, f.pluginConfig, logger, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.eventBus, f.tmplMgr).(*verifyEmailUseCase)
		err := uc.handleEmailChangeVerificationEmail(testRequestContext(), emailChangeUserID, "token-123", emailChangeNewEmail)

		require.NoError(t, err)
		require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send changed email to new email via plugin callback")
	})
}

func TestVerifyEmailUseCase_SendChangedEmailEmails(t *testing.T) {
	t.Parallel()

	t.Run("sends a notification to the old and new address", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		f.mailerSvc.On("SendEmail", mock.Anything, emailChangeOldEmail, "Your email has been changed", mock.Anything, mock.Anything).Return(nil).Once()
		f.mailerSvc.On("SendEmail", mock.Anything, emailChangeNewEmail, "Your email has been changed", mock.Anything, mock.Anything).Return(nil).Once()

		uc := f.verifyEmailUseCase().(*verifyEmailUseCase)
		err := uc.sendChangedEmailEmails(context.Background(), emailChangeOldEmail, emailChangeNewEmail)

		require.NoError(t, err)
		f.mailerSvc.AssertExpectations(t)
	})

	t.Run("render errors are logged and skipped", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		logger := newEmailChangeNotifyingLogger()

		uc := &verifyEmailUseCase{
			GlobalConfig:         f.globalConfig,
			PluginConfig:         f.pluginConfig,
			Logger:               logger,
			EmailTemplateManager: emailtmpl.NewManager(),
		}
		err := uc.sendChangedEmailEmails(context.Background(), emailChangeOldEmail, emailChangeNewEmail)

		require.NoError(t, err)
		require.Len(t, logger.messages(), 2)
		require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to render email changed notification template")
	})

	t.Run("send errors are logged", func(t *testing.T) {
		t.Parallel()
		f := newEmailChangeFixture(t)
		logger := newEmailChangeNotifyingLogger()
		f.mailerSvc.On("SendEmail", mock.Anything, emailChangeOldEmail, mock.Anything, mock.Anything, mock.Anything).Return(errEmailChangeBackend).Once()
		f.mailerSvc.On("SendEmail", mock.Anything, emailChangeNewEmail, mock.Anything, mock.Anything, mock.Anything).Return(errEmailChangeBackend).Once()

		uc := NewVerifyEmailUseCase(f.globalConfig, f.pluginConfig, logger, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.eventBus, f.tmplMgr).(*verifyEmailUseCase)
		err := uc.sendChangedEmailEmails(context.Background(), emailChangeOldEmail, emailChangeNewEmail)

		require.NoError(t, err)
		require.Len(t, logger.messages(), 2)
		require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send email to address via built-in email service")
		f.mailerSvc.AssertExpectations(t)
	})
}
