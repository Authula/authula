package tests

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func failUnexpected(t *testing.T, strict bool, method string) {
	if !strict || t == nil {
		return
	}
	t.Helper()
	t.Fatalf("unexpected call to %s", method)
}

type MockUserService struct {
	t              *testing.T
	strict         bool
	GetByEmailFn   func(ctx context.Context, email string) (*models.User, error)
	GetByIDFn      func(ctx context.Context, id string) (*models.User, error)
	CreateFn       func(ctx context.Context, name, email string, emailVerified bool, image *string) (*models.User, error)
	UpdateFn       func(ctx context.Context, user *models.User) (*models.User, error)
	UpdateFieldsFn func(ctx context.Context, id string, fields map[string]any) error
}

func NewMockUserService(t *testing.T) *MockUserService {
	t.Helper()
	return &MockUserService{t: t, strict: true}
}

func (m *MockUserService) Create(ctx context.Context, name string, email string, emailVerified bool, image *string) (*models.User, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, name, email, emailVerified, image)
	}
	failUnexpected(m.t, m.strict, "MockUserService.Create")
	return &models.User{ID: "user-1", Name: name, Email: email, EmailVerified: emailVerified, Image: image}, nil
}

func (m *MockUserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockUserService.GetByID")
	return &models.User{ID: id, Email: "test@example.com"}, nil
}

func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	failUnexpected(m.t, m.strict, "MockUserService.GetByEmail")
	return &models.User{ID: "user-1", Email: email}, nil
}

func (m *MockUserService) Update(ctx context.Context, user *models.User) (*models.User, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	failUnexpected(m.t, m.strict, "MockUserService.Update")
	return user, nil
}

func (m *MockUserService) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	if m.UpdateFieldsFn != nil {
		return m.UpdateFieldsFn(ctx, id, fields)
	}
	failUnexpected(m.t, m.strict, "MockUserService.UpdateFields")
	return nil
}

type MockAccountService struct {
	t                           *testing.T
	strict                      bool
	CreateFn                    func(ctx context.Context, userID, accountID, providerID string, password *string) (*models.Account, error)
	CreateOAuth2Fn              func(ctx context.Context, userID, providerAccountID, provider, accessToken string, refreshToken *string, accessTokenExpiresAt, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error)
	GetByUserIDFn               func(ctx context.Context, userID string) (*models.Account, error)
	GetByUserIDAndProviderFn    func(ctx context.Context, userID, provider string) (*models.Account, error)
	GetByProviderAndAccountIDFn func(ctx context.Context, provider, accountID string) (*models.Account, error)
	UpdateFn                    func(ctx context.Context, account *models.Account) (*models.Account, error)
	UpdateFieldsFn              func(ctx context.Context, userID string, fields map[string]any) error
}

func NewMockAccountService(t *testing.T) *MockAccountService {
	t.Helper()
	return &MockAccountService{t: t, strict: true}
}

func (m *MockAccountService) Create(ctx context.Context, userID, accountID, providerID string, password *string) (*models.Account, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, accountID, providerID, password)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.Create")
	return &models.Account{ID: "account-1", UserID: userID}, nil
}

func (m *MockAccountService) CreateOAuth2(ctx context.Context, userID, providerAccountID, provider, accessToken string, refreshToken *string, accessTokenExpiresAt, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	if m.CreateOAuth2Fn != nil {
		return m.CreateOAuth2Fn(ctx, userID, providerAccountID, provider, accessToken, refreshToken, accessTokenExpiresAt, refreshTokenExpiresAt, scope)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.CreateOAuth2")
	return nil, nil
}

func (m *MockAccountService) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.GetByUserID")
	return nil, nil
}

func (m *MockAccountService) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.Account, error) {
	if m.GetByUserIDAndProviderFn != nil {
		return m.GetByUserIDAndProviderFn(ctx, userID, provider)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.GetByUserIDAndProvider")
	return nil, nil
}

func (m *MockAccountService) GetByProviderAndAccountID(ctx context.Context, provider, accountID string) (*models.Account, error) {
	if m.GetByProviderAndAccountIDFn != nil {
		return m.GetByProviderAndAccountIDFn(ctx, provider, accountID)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.GetByProviderAndAccountID")
	return nil, nil
}

func (m *MockAccountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, account)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.Update")
	return account, nil
}

func (m *MockAccountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	if m.UpdateFieldsFn != nil {
		return m.UpdateFieldsFn(ctx, userID, fields)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.UpdateFields")
	return nil
}

type MockSessionService struct {
	t                           *testing.T
	strict                      bool
	GetByIDFn                   func(ctx context.Context, id string) (*models.Session, error)
	CreateFn                    func(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error)
	GetByUserIDFn               func(ctx context.Context, userID string) (*models.Session, error)
	GetByTokenFn                func(ctx context.Context, hashedToken string) (*models.Session, error)
	UpdateFn                    func(ctx context.Context, session *models.Session) (*models.Session, error)
	DeleteFn                    func(ctx context.Context, id string) error
	DeleteAllByUserIDFn         func(ctx context.Context, userID string) error
	CleanupExpiredSessionsFn    func(ctx context.Context) error
	EnforceMaxSessionsPerUserFn func(ctx context.Context, maxPerUser int) error
	RunCleanupFn                func(ctx context.Context, maxPerUser int) error
}

func NewMockSessionService(t *testing.T) *MockSessionService {
	t.Helper()
	return &MockSessionService{t: t, strict: true}
}

func (m *MockSessionService) GetByID(ctx context.Context, id string) (*models.Session, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetByID")
	return nil, nil
}

func (m *MockSessionService) Create(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, hashedToken, ipAddress, userAgent, maxAge)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.Create")
	return &models.Session{ID: "session-1", UserID: userID, IPAddress: ipAddress, UserAgent: userAgent, ExpiresAt: time.Now().Add(maxAge)}, nil
}

func (m *MockSessionService) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetByUserID")
	return nil, nil
}

func (m *MockSessionService) GetByToken(ctx context.Context, hashedToken string) (*models.Session, error) {
	if m.GetByTokenFn != nil {
		return m.GetByTokenFn(ctx, hashedToken)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetByToken")
	return nil, nil
}

func (m *MockSessionService) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, session)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.Update")
	return session, nil
}

func (m *MockSessionService) Delete(ctx context.Context, id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.Delete")
	return nil
}

func (m *MockSessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	if m.DeleteAllByUserIDFn != nil {
		return m.DeleteAllByUserIDFn(ctx, userID)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.DeleteAllByUserID")
	return nil
}

func (m *MockSessionService) CleanupExpiredSessions(ctx context.Context) error {
	if m.CleanupExpiredSessionsFn != nil {
		return m.CleanupExpiredSessionsFn(ctx)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.CleanupExpiredSessions")
	return nil
}

func (m *MockSessionService) EnforceMaxSessionsPerUser(ctx context.Context, maxPerUser int) error {
	if m.EnforceMaxSessionsPerUserFn != nil {
		return m.EnforceMaxSessionsPerUserFn(ctx, maxPerUser)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.EnforceMaxSessionsPerUser")
	return nil
}

func (m *MockSessionService) RunCleanup(ctx context.Context, maxPerUser int) error {
	if m.RunCleanupFn != nil {
		return m.RunCleanupFn(ctx, maxPerUser)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.RunCleanup")
	return nil
}

type MockVerificationService struct {
	t                       *testing.T
	strict                  bool
	CreateFn                func(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error)
	GetByTokenFn            func(ctx context.Context, hashedToken string) (*models.Verification, error)
	DeleteFn                func(ctx context.Context, id string) error
	DeleteByUserIDAndTypeFn func(ctx context.Context, userID string, vType models.VerificationType) error
	IsExpiredFn             func(verif *models.Verification) bool
}

func NewMockVerificationService(t *testing.T) *MockVerificationService {
	t.Helper()
	return &MockVerificationService{t: t, strict: true}
}

func (m *MockVerificationService) Create(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, hashedToken, vType, value, expiry)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.Create")
	return &models.Verification{ID: "verification-1", UserID: &userID, Identifier: value, Type: vType, ExpiresAt: time.Now().Add(expiry)}, nil
}

func (m *MockVerificationService) GetByToken(ctx context.Context, hashedToken string) (*models.Verification, error) {
	if m.GetByTokenFn != nil {
		return m.GetByTokenFn(ctx, hashedToken)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.GetByToken")
	return nil, nil
}

func (m *MockVerificationService) Delete(ctx context.Context, id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.Delete")
	return nil
}

func (m *MockVerificationService) DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error {
	if m.DeleteByUserIDAndTypeFn != nil {
		return m.DeleteByUserIDAndTypeFn(ctx, userID, vType)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.DeleteByUserIDAndType")
	return nil
}

func (m *MockVerificationService) IsExpired(verif *models.Verification) bool {
	if m.IsExpiredFn != nil {
		return m.IsExpiredFn(verif)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.IsExpired")
	return false
}

type MockTokenService struct {
	t          *testing.T
	strict     bool
	GenerateFn func() (string, error)
	HashFn     func(token string) string
	EncryptFn  func(token string) (string, error)
	DecryptFn  func(encrypted string) (string, error)
}

func NewMockTokenService(t *testing.T) *MockTokenService {
	t.Helper()
	return &MockTokenService{t: t, strict: true}
}

func (m *MockTokenService) Generate() (string, error) {
	if m.GenerateFn != nil {
		return m.GenerateFn()
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Generate")
	return "test-token-123", nil
}

func (m *MockTokenService) Hash(token string) string {
	if m.HashFn != nil {
		return m.HashFn(token)
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Hash")
	return "hashed-" + token
}

func (m *MockTokenService) Encrypt(token string) (string, error) {
	if m.EncryptFn != nil {
		return m.EncryptFn(token)
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Encrypt")
	return token, nil
}

func (m *MockTokenService) Decrypt(encrypted string) (string, error) {
	if m.DecryptFn != nil {
		return m.DecryptFn(encrypted)
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Decrypt")
	return encrypted, nil
}

type MockMailerService struct {
	t           *testing.T
	strict      bool
	SendEmailFn func(ctx context.Context, to string, subject string, text string, html string) error
}

func NewMockMailerService(t *testing.T) *MockMailerService {
	t.Helper()
	return &MockMailerService{t: t, strict: true}
}

func (m *MockMailerService) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	if m.SendEmailFn != nil {
		return m.SendEmailFn(ctx, to, subject, text, html)
	}
	failUnexpected(m.t, m.strict, "MockMailerService.SendEmail")
	return nil
}

type MockLogger struct{}

func (m *MockLogger) Debug(msg string, args ...any) {}
func (m *MockLogger) Info(msg string, args ...any)  {}
func (m *MockLogger) Warn(msg string, args ...any)  {}
func (m *MockLogger) Error(msg string, args ...any) {}
func (m *MockLogger) Panic(msg string, args ...any) {}
func (m *MockLogger) WithField(key string, value any) models.Logger {
	return m
}
func (m *MockLogger) WithFields(fields map[string]any) models.Logger {
	return m
}
