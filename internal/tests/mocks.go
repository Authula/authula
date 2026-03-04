package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func failUnexpected(t *testing.T, strict bool, method string) {
	if !strict || t == nil {
		return
	}
	t.Helper()
	t.Fatalf("unexpected call to %s", method)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error) {
	args := m.Called(ctx, cursor, limit)
	users := args.Get(0)
	cursor2 := args.Get(1)

	var usersSlice []models.User
	if users != nil {
		usersSlice = users.([]models.User)
	}

	var cursorPtr *string
	if cursor2 != nil {
		cursorPtr = cursor2.(*string)
	}

	return usersSlice, cursorPtr, args.Error(2)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, name string, email string, emailVerified bool, image *string, metadata json.RawMessage) (*models.User, error) {
	args := m.Called(ctx, name, email, emailVerified, image, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockAccountService struct {
	mock.Mock
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
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, accountID, providerID, password)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Account), args.Error(1)
	}
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, accountID, providerID, password)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.Create")
	return &models.Account{ID: "account-1", UserID: userID}, nil
}

func (m *MockAccountService) CreateOAuth2(ctx context.Context, userID, providerAccountID, provider, accessToken string, refreshToken *string, accessTokenExpiresAt, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, providerAccountID, provider, accessToken, refreshToken, accessTokenExpiresAt, refreshTokenExpiresAt, scope)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Account), args.Error(1)
	}
	if m.CreateOAuth2Fn != nil {
		return m.CreateOAuth2Fn(ctx, userID, providerAccountID, provider, accessToken, refreshToken, accessTokenExpiresAt, refreshTokenExpiresAt, scope)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.CreateOAuth2")
	return nil, nil
}

func (m *MockAccountService) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Account), args.Error(1)
	}
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.GetByUserID")
	return nil, nil
}

func (m *MockAccountService) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.Account, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, provider)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Account), args.Error(1)
	}
	if m.GetByUserIDAndProviderFn != nil {
		return m.GetByUserIDAndProviderFn(ctx, userID, provider)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.GetByUserIDAndProvider")
	return nil, nil
}

func (m *MockAccountService) GetByProviderAndAccountID(ctx context.Context, provider, accountID string) (*models.Account, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, provider, accountID)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Account), args.Error(1)
	}
	if m.GetByProviderAndAccountIDFn != nil {
		return m.GetByProviderAndAccountIDFn(ctx, provider, accountID)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.GetByProviderAndAccountID")
	return nil, nil
}

func (m *MockAccountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, account)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Account), args.Error(1)
	}
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, account)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.Update")
	return account, nil
}

func (m *MockAccountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, fields)
		return args.Error(0)
	}
	if m.UpdateFieldsFn != nil {
		return m.UpdateFieldsFn(ctx, userID, fields)
	}
	failUnexpected(m.t, m.strict, "MockAccountService.UpdateFields")
	return nil
}

type MockSessionService struct {
	mock.Mock
	t                      *testing.T
	strict                 bool
	GetByIDFn              func(ctx context.Context, id string) (*models.Session, error)
	CreateFn               func(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error)
	GetByUserIDFn          func(ctx context.Context, userID string) (*models.Session, error)
	GetByTokenFn           func(ctx context.Context, hashedToken string) (*models.Session, error)
	UpdateFn               func(ctx context.Context, session *models.Session) (*models.Session, error)
	DeleteFn               func(ctx context.Context, id string) error
	DeleteAllByUserIDFn    func(ctx context.Context, userID string) error
	DeleteAllExpiredFn     func(ctx context.Context) error
	GetDistinctUserIDsFn   func(ctx context.Context) ([]string, error)
	DeleteOldestByUserIDFn func(ctx context.Context, userID string, maxCount int) error
}

func NewMockSessionService(t *testing.T) *MockSessionService {
	t.Helper()
	return &MockSessionService{t: t, strict: true}
}

func (m *MockSessionService) GetByID(ctx context.Context, id string) (*models.Session, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, id)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Session), args.Error(1)
	}
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetByID")
	return nil, nil
}

func (m *MockSessionService) Create(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, hashedToken, ipAddress, userAgent, maxAge)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Session), args.Error(1)
	}
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, hashedToken, ipAddress, userAgent, maxAge)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.Create")
	return &models.Session{ID: "session-1", UserID: userID, IPAddress: ipAddress, UserAgent: userAgent, ExpiresAt: time.Now().Add(maxAge)}, nil
}

func (m *MockSessionService) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Session), args.Error(1)
	}
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetByUserID")
	return nil, nil
}

func (m *MockSessionService) GetByToken(ctx context.Context, hashedToken string) (*models.Session, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, hashedToken)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Session), args.Error(1)
	}
	if m.GetByTokenFn != nil {
		return m.GetByTokenFn(ctx, hashedToken)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetByToken")
	return nil, nil
}

func (m *MockSessionService) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, session)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Session), args.Error(1)
	}
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, session)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.Update")
	return session, nil
}

func (m *MockSessionService) Delete(ctx context.Context, id string) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, id)
		return args.Error(0)
	}
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.Delete")
	return nil
}

func (m *MockSessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID)
		return args.Error(0)
	}
	if m.DeleteAllByUserIDFn != nil {
		return m.DeleteAllByUserIDFn(ctx, userID)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.DeleteAllByUserID")
	return nil
}

func (m *MockSessionService) DeleteAllExpired(ctx context.Context) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx)
		return args.Error(0)
	}
	if m.DeleteAllExpiredFn != nil {
		return m.DeleteAllExpiredFn(ctx)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.DeleteAllExpired")
	return nil
}

func (m *MockSessionService) GetDistinctUserIDs(ctx context.Context) ([]string, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).([]string), args.Error(1)
	}
	if m.GetDistinctUserIDsFn != nil {
		return m.GetDistinctUserIDsFn(ctx)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.GetDistinctUserIDs")
	return nil, nil
}

func (m *MockSessionService) DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, maxCount)
		return args.Error(0)
	}
	if m.DeleteOldestByUserIDFn != nil {
		return m.DeleteOldestByUserIDFn(ctx, userID, maxCount)
	}
	failUnexpected(m.t, m.strict, "MockSessionService.DeleteOldestByUserID")
	return nil
}

type MockVerificationService struct {
	mock.Mock
	t                       *testing.T
	strict                  bool
	CreateFn                func(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error)
	GetByTokenFn            func(ctx context.Context, hashedToken string) (*models.Verification, error)
	DeleteFn                func(ctx context.Context, id string) error
	DeleteByUserIDAndTypeFn func(ctx context.Context, userID string, vType models.VerificationType) error
	IsExpiredFn             func(verif *models.Verification) bool
	DeleteExpiredFn         func(ctx context.Context) error
}

func NewMockVerificationService(t *testing.T) *MockVerificationService {
	t.Helper()
	return &MockVerificationService{t: t, strict: true}
}

func (m *MockVerificationService) Create(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, hashedToken, vType, value, expiry)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Verification), args.Error(1)
	}
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, hashedToken, vType, value, expiry)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.Create")
	return &models.Verification{ID: "verification-1", UserID: &userID, Identifier: value, Type: vType, ExpiresAt: time.Now().Add(expiry)}, nil
}

func (m *MockVerificationService) GetByToken(ctx context.Context, hashedToken string) (*models.Verification, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, hashedToken)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*models.Verification), args.Error(1)
	}
	if m.GetByTokenFn != nil {
		return m.GetByTokenFn(ctx, hashedToken)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.GetByToken")
	return nil, nil
}

func (m *MockVerificationService) Delete(ctx context.Context, id string) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, id)
		return args.Error(0)
	}
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.Delete")
	return nil
}

func (m *MockVerificationService) DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, userID, vType)
		return args.Error(0)
	}
	if m.DeleteByUserIDAndTypeFn != nil {
		return m.DeleteByUserIDAndTypeFn(ctx, userID, vType)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.DeleteByUserIDAndType")
	return nil
}

func (m *MockVerificationService) IsExpired(verif *models.Verification) bool {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(verif)
		return args.Bool(0)
	}
	if m.IsExpiredFn != nil {
		return m.IsExpiredFn(verif)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.IsExpired")
	return false
}

func (m *MockVerificationService) DeleteExpired(ctx context.Context) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx)
		return args.Error(0)
	}
	if m.DeleteExpiredFn != nil {
		return m.DeleteExpiredFn(ctx)
	}
	failUnexpected(m.t, m.strict, "MockVerificationService.DeleteExpired")
	return nil
}

type MockTokenService struct {
	mock.Mock
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
	if len(m.ExpectedCalls) > 0 {
		args := m.Called()
		if args.Get(0) == nil {
			return "", args.Error(1)
		}
		return args.String(0), args.Error(1)
	}
	if m.GenerateFn != nil {
		return m.GenerateFn()
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Generate")
	return "test-token-123", nil
}

func (m *MockTokenService) Hash(token string) string {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(token)
		return args.String(0)
	}
	if m.HashFn != nil {
		return m.HashFn(token)
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Hash")
	return "hashed-" + token
}

func (m *MockTokenService) Encrypt(token string) (string, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(token)
		if args.Get(0) == nil {
			return "", args.Error(1)
		}
		return args.String(0), args.Error(1)
	}
	if m.EncryptFn != nil {
		return m.EncryptFn(token)
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Encrypt")
	return token, nil
}

func (m *MockTokenService) Decrypt(encrypted string) (string, error) {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(encrypted)
		if args.Get(0) == nil {
			return "", args.Error(1)
		}
		return args.String(0), args.Error(1)
	}
	if m.DecryptFn != nil {
		return m.DecryptFn(encrypted)
	}
	failUnexpected(m.t, m.strict, "MockTokenService.Decrypt")
	return encrypted, nil
}

type MockMailerService struct {
	mock.Mock
	t           *testing.T
	strict      bool
	SendEmailFn func(ctx context.Context, to string, subject string, text string, html string) error
}

func NewMockMailerService(t *testing.T) *MockMailerService {
	t.Helper()
	return &MockMailerService{t: t, strict: true}
}

func (m *MockMailerService) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	if len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, to, subject, text, html)
		return args.Error(0)
	}
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
