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

var errSendVerificationBackend = errors.New("database unavailable")

// notifyingLogger records error messages and signals the first one, so tests
// can wait for logging that happens in a background goroutine.
type notifyingLogger struct {
	mu     sync.Mutex
	errors []string
	done   chan struct{}
	once   sync.Once
}

func newNotifyingLogger() *notifyingLogger {
	return &notifyingLogger{done: make(chan struct{})}
}

func (l *notifyingLogger) Debug(string, ...any) {}
func (l *notifyingLogger) Info(string, ...any)  {}
func (l *notifyingLogger) Warn(string, ...any)  {}

func (l *notifyingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.errors = append(l.errors, msg)
	l.mu.Unlock()
	l.once.Do(func() { close(l.done) })
}

func (l *notifyingLogger) messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.errors...)
}

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestSendEmailVerificationUseCase_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*emailPasswordTestFixture)
		wantErr string
		wantLog string
	}{
		{
			name: "user lookup error is logged and swallowed",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, errSendVerificationBackend).Once()
			},
			wantLog: "failed to fetch user",
		},
		{
			name: "token generation error is wrapped and returned",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("", errSendVerificationBackend).Once()
			},
			wantErr: "failed to generate verification token",
			wantLog: "database unavailable",
		},
		{
			name: "delete failure is logged and execution continues",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("verify-token", nil).Once()
				f.tokenSvc.On("Hash", "verify-token").Return("hashed-token").Once()
				f.verificationSvc.On("DeleteByUserIDAndType", mock.Anything, "user-1", models.TypeEmailVerification).Return(errSendVerificationBackend).Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailVerification, "test@example.com", 24*time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
			},
			wantLog: "database unavailable",
		},
		{
			name: "create failure is wrapped and returned",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("verify-token", nil).Once()
				f.tokenSvc.On("Hash", "verify-token").Return("hashed-token").Once()
				f.verificationSvc.On("DeleteByUserIDAndType", mock.Anything, "user-1", models.TypeEmailVerification).Return(nil).Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailVerification, "test@example.com", 24*time.Hour).Return(nil, errSendVerificationBackend).Once()
			},
			wantErr: "failed to create verification token",
			wantLog: "database unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newEmailPasswordTestFixture()
			tc.setup(f)

			// An untyped nil mailer keeps these cases free of the background
			// goroutine. Passing the typed nil mock would still satisfy the
			// interface and panic inside the goroutine.
			logger := newNotifyingLogger()
			uc := NewSendEmailVerificationUseCase(f.globalConfig, f.pluginConfig, logger, f.userSvc, f.verificationSvc, f.tokenSvc, nil, f.tmplMgr)

			err := uc.Send(testRequestContext(), "user-1", nil)

			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			if tc.wantLog != "" {
				require.Contains(t, strings.Join(logger.messages(), "\n"), tc.wantLog)
			}
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestSendEmailVerificationUseCase_MailerFallback(t *testing.T) {
	t.Parallel()

	expectVerificationCreated := func(f *emailPasswordTestFixture) {
		f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
		f.tokenSvc.On("Generate").Return("verify-token", nil).Once()
		f.tokenSvc.On("Hash", "verify-token").Return("hashed-token").Once()
		f.verificationSvc.On("DeleteByUserIDAndType", mock.Anything, "user-1", models.TypeEmailVerification).Return(nil).Once()
		f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailVerification, "test@example.com", 24*time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
	}

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture, chan struct{})
		assert func(*testing.T, *emailPasswordTestFixture, *notifyingLogger, <-chan struct{})
	}{
		{
			name: "plugin callback failure falls back to the mailer",
			setup: func(f *emailPasswordTestFixture, sent chan struct{}) {
				expectVerificationCreated(f)
				f.pluginConfig.SendEmailVerification = func(types.SendEmailVerificationParams, *models.RequestContext) error {
					return errSendVerificationBackend
				}
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", "Verify your email", mock.Anything, mock.Anything).
					Run(func(mock.Arguments) { close(sent) }).
					Return(nil).Once()
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, logger *notifyingLogger, sent <-chan struct{}) {
				waitForSignal(t, sent, "mailer send")
				require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send email verification via plugin callback")
				f.mailerSvc.AssertExpectations(t)
			},
		},
		{
			name: "plugin callback success skips the mailer",
			setup: func(f *emailPasswordTestFixture, _ chan struct{}) {
				expectVerificationCreated(f)
				f.pluginConfig.SendEmailVerification = func(types.SendEmailVerificationParams, *models.RequestContext) error {
					return nil
				}
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, _ *notifyingLogger, _ <-chan struct{}) {
				f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
			},
		},
		{
			name: "render failure in the background is logged",
			setup: func(f *emailPasswordTestFixture, _ chan struct{}) {
				expectVerificationCreated(f)
				// A manager without registered templates forces the render path to fail.
				f.tmplMgr = emailtmpl.NewManager()
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, logger *notifyingLogger, _ <-chan struct{}) {
				waitForSignal(t, logger.done, "background render error")
				require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send email verification via built-in email service")
				f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newEmailPasswordTestFixture()
			logger := newNotifyingLogger()
			sent := make(chan struct{})
			tc.setup(f, sent)

			uc := NewSendEmailVerificationUseCase(f.globalConfig, f.pluginConfig, logger, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr)
			require.NoError(t, uc.Send(testRequestContext(), "user-1", nil))

			tc.assert(t, f, logger, sent)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestSendEmailVerificationUseCase_SendEmailVerification(t *testing.T) {
	t.Parallel()

	const link = "http://localhost/auth/verify?token=abc"
	user := &models.User{ID: "user-1", Email: "test@example.com"}

	tests := []struct {
		name    string
		setup   func(*emailPasswordTestFixture)
		wantErr string
		wantLog string
	}{
		{
			name: "renders the template and sends it",
			setup: func(f *emailPasswordTestFixture) {
				f.mailerSvc.On(
					"SendEmail",
					mock.Anything,
					"test@example.com",
					"Verify your email",
					mock.MatchedBy(func(body string) bool { return strings.Contains(body, link) }),
					mock.MatchedBy(func(body string) bool { return strings.Contains(body, link) }),
				).Return(nil).Once()
			},
		},
		{
			name: "render failure is logged and returned",
			setup: func(f *emailPasswordTestFixture) {
				// A manager without registered templates forces the render path to fail.
				f.tmplMgr = emailtmpl.NewManager()
			},
			wantErr: "not found",
			wantLog: "failed to render email verification template",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newEmailPasswordTestFixture()
			logger := newNotifyingLogger()
			tc.setup(f)

			uc := NewSendEmailVerificationUseCase(f.globalConfig, f.pluginConfig, logger, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr).(*sendEmailVerificationUseCase)
			err := uc.sendEmailVerification(context.Background(), user, link)

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
