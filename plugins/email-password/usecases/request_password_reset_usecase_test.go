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

func TestRequestPasswordResetUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name: "missing user returns nil",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "creates reset verification",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("reset-token", nil).Once()
				f.tokenSvc.On("Hash", "reset-token").Return("hashed-token").Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypePasswordResetRequest, "test@example.com", time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
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
			err := f.requestPasswordResetUseCase().RequestReset(testRequestContext(), "test@example.com", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

var errPasswordFlowBackend = errors.New("database unavailable")

// passwordFlowLogger records error messages and signals the first one, so tests
// can wait for logging that happens in a background goroutine.
type passwordFlowLogger struct {
	mu     sync.Mutex
	errors []string
	done   chan struct{}
	once   sync.Once
}

func newPasswordFlowLogger() *passwordFlowLogger {
	return &passwordFlowLogger{done: make(chan struct{})}
}

func (l *passwordFlowLogger) Debug(string, ...any) {}
func (l *passwordFlowLogger) Info(string, ...any)  {}
func (l *passwordFlowLogger) Warn(string, ...any)  {}

func (l *passwordFlowLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.errors = append(l.errors, msg)
	l.mu.Unlock()
	l.once.Do(func() { close(l.done) })
}

func (l *passwordFlowLogger) messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.errors...)
}

func waitForPasswordFlowSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

// newPasswordResetFixture registers the reset template under the name the
// production template definitions use. The shared fixture registers
// "password_reset", which the renderer never looks up.
func newPasswordResetFixture(t *testing.T) *emailPasswordTestFixture {
	t.Helper()
	f := newEmailPasswordTestFixture()
	require.NoError(t, f.tmplMgr.Register(emailtmpl.Definition{
		Name:    "password_reset_request",
		Subject: "Reset Your Password",
		Text:    "Reset link: {{.ResetLink}}",
		HTML:    "<p>Reset: {{.ResetLink}}</p>",
	}))
	return f
}

func TestRequestPasswordResetUseCase_TokenAndVerificationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setup          func(*emailPasswordTestFixture)
		expectErr      error
		expectContains string
	}{
		{
			name: "token generation failure is wrapped",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("", errPasswordFlowBackend).Once()
			},
			expectContains: "failed to generate reset token",
		},
		{
			name: "verification create failure is swallowed to avoid enumeration",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("reset-token", nil).Once()
				f.tokenSvc.On("Hash", "reset-token").Return("hashed-token").Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypePasswordResetRequest, "test@example.com", time.Hour).Return(nil, errPasswordFlowBackend).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newPasswordResetFixture(t)
			tt.setup(f)

			err := f.requestPasswordResetUseCase().RequestReset(testRequestContext(), "test@example.com", nil)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else if tt.expectContains != "" {
				require.ErrorContains(t, err, tt.expectContains)
			} else {
				require.NoError(t, err)
			}
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestRequestPasswordResetUseCase_PluginCallback(t *testing.T) {
	t.Parallel()

	expectVerification := func(f *emailPasswordTestFixture) {
		f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
		f.tokenSvc.On("Generate").Return("reset-token", nil).Once()
		f.tokenSvc.On("Hash", "reset-token").Return("hashed-token").Once()
		f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypePasswordResetRequest, "test@example.com", time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
	}

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture, chan struct{})
		assert func(*testing.T, *emailPasswordTestFixture, *passwordFlowLogger, <-chan struct{})
	}{
		{
			name: "successful callback skips the mailer",
			setup: func(f *emailPasswordTestFixture, _ chan struct{}) {
				expectVerification(f)
				f.pluginConfig.SendPasswordResetEmail = func(types.SendPasswordResetEmailParams, *models.RequestContext) error {
					return nil
				}
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, _ *passwordFlowLogger, _ <-chan struct{}) {
				f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
			},
		},
		{
			name: "callback failure is logged and falls back to the mailer",
			setup: func(f *emailPasswordTestFixture, sent chan struct{}) {
				expectVerification(f)
				f.pluginConfig.SendPasswordResetEmail = func(types.SendPasswordResetEmailParams, *models.RequestContext) error {
					return errPasswordFlowBackend
				}
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", "Reset Your Password", mock.Anything, mock.Anything).
					Run(func(mock.Arguments) { close(sent) }).
					Return(nil).Once()
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, logger *passwordFlowLogger, sent <-chan struct{}) {
				waitForPasswordFlowSignal(t, sent, "password reset email")
				require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send password reset email via plugin callback")
				f.mailerSvc.AssertExpectations(t)
			},
		},
		{
			name: "render failure in the background is logged",
			setup: func(f *emailPasswordTestFixture, _ chan struct{}) {
				expectVerification(f)
				// A manager without registered templates forces the render path to fail.
				f.tmplMgr = emailtmpl.NewManager()
			},
			assert: func(t *testing.T, f *emailPasswordTestFixture, logger *passwordFlowLogger, _ <-chan struct{}) {
				waitForPasswordFlowSignal(t, logger.done, "background render error")
				require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send password reset email via built-in email service")
				f.mailerSvc.AssertNumberOfCalls(t, "SendEmail", 0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newPasswordResetFixture(t)
			logger := newPasswordFlowLogger()
			sent := make(chan struct{}, 1)
			tc.setup(f, sent)

			uc := NewRequestPasswordResetUseCase(logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr, f.hooksExecutor)
			err := uc.RequestReset(testRequestContext(), "test@example.com", nil)

			require.NoError(t, err)
			tc.assert(t, f, logger, sent)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestRequestPasswordResetUseCase_SendRequestPasswordResetEmail(t *testing.T) {
	t.Parallel()

	const (
		userEmail = "test@example.com"
		subject   = "Reset Your Password"
		link      = "http://localhost/auth/reset?token=abc"
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
				f.mailerSvc.On(
					"SendEmail",
					mock.Anything,
					userEmail,
					subject,
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
			wantLog: "failed to render password reset template",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newPasswordResetFixture(t)
			logger := newPasswordFlowLogger()
			tc.setup(f)

			uc := NewRequestPasswordResetUseCase(logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr, f.hooksExecutor).(*requestPasswordResetUseCase)
			err := uc.sendRequestPasswordResetEmail(context.Background(), &models.User{ID: "user-1", Email: userEmail}, link)

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
