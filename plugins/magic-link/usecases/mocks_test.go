package usecases

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	emailconstants "github.com/Authula/authula/core/email/constants"
	emailtmpl "github.com/Authula/authula/core/email/template"
	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
)

type mockUserService = inttests.MockUserService
type mockAccountService = inttests.MockAccountService
type mockSessionService = inttests.MockSessionService
type mockVerificationService = inttests.MockVerificationService
type mockTokenService = inttests.MockTokenService
type mockMailerService = inttests.MockMailerService
type mockLogger = inttests.MockLogger

func newMockUserService() *mockUserService {
	return &mockUserService{}
}

func newMockAccountService() *mockAccountService {
	return &mockAccountService{}
}

func newMockTokenService() *mockTokenService {
	return &mockTokenService{}
}

func newMockVerificationService() *mockVerificationService {
	return &mockVerificationService{}
}

func newMockMailerService() *mockMailerService {
	return &mockMailerService{}
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{}
}

func newSignInTestUseCase() (*SignInUseCaseImpl, *mockUserService, *mockAccountService, *mockTokenService, *mockVerificationService, *mockMailerService) {
	userSvc := newMockUserService()
	accountSvc := newMockAccountService()
	tokenSvc := newMockTokenService()
	verificationSvc := newMockVerificationService()
	mailerSvc := newMockMailerService()

	uc := &SignInUseCaseImpl{
		GlobalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
			AppName:  "TestApp",
		},
		PluginConfig: &types.MagicLinkPluginConfig{
			ExpiresIn: 15 * time.Minute,
		},
		Logger:              &mockLogger{},
		UserService:         userSvc,
		AccountService:      accountSvc,
		TokenService:        tokenSvc,
		VerificationService: verificationSvc,
		MailerService:       nil,
	}

	return uc, userSvc, accountSvc, tokenSvc, verificationSvc, mailerSvc
}

func newVerifyTestUseCase() (*VerifyUseCaseImpl, *mockUserService, *mockVerificationService, *mockTokenService) {
	userSvc := newMockUserService()
	verificationSvc := newMockVerificationService()
	tokenSvc := newMockTokenService()

	uc := &VerifyUseCaseImpl{
		GlobalConfig: &models.Config{
			Session: models.SessionConfig{ExpiresIn: 24 * time.Hour},
		},
		PluginConfig:        &types.MagicLinkPluginConfig{ExpiresIn: 15 * time.Minute},
		Logger:              &mockLogger{},
		UserService:         userSvc,
		VerificationService: verificationSvc,
		TokenService:        tokenSvc,
	}

	return uc, userSvc, verificationSvc, tokenSvc
}

// newSignInTemplateManager builds a template manager that can render the
// magic-link sign-in email without loading the plugin's embedded templates.
func newSignInTemplateManager(t *testing.T) *emailtmpl.Manager {
	t.Helper()

	mgr := emailtmpl.NewManager()
	require.NoError(t, mgr.Register(emailtmpl.Definition{
		Name:    emailconstants.MagicLinkSignInEmailTemplateName,
		Subject: "Sign in to {{.AppName}}",
		Text:    "Open this link to sign in: {{.MagicLink}}",
		HTML:    `<p>Open <a href="{{.MagicLink}}">this link</a> to sign in.</p>`,
	}))

	return mgr
}

// errorSignalingLogger reports error messages and signals when the first one
// arrives, so tests can wait for work performed inside a goroutine.
type errorSignalingLogger struct {
	mockLogger

	mu       sync.Mutex
	messages []string
	first    sync.Once
	done     chan struct{}
}

func newErrorSignalingLogger() *errorSignalingLogger {
	return &errorSignalingLogger{done: make(chan struct{})}
}

func (l *errorSignalingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.messages = append(l.messages, msg)
	l.mu.Unlock()

	l.first.Do(func() { close(l.done) })
}

func (l *errorSignalingLogger) messagesSnapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]string(nil), l.messages...)
}
