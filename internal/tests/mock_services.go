package tests

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(ctx context.Context, name string, email string, emailVerified bool, image *string, metadata json.RawMessage) (*models.User, error) {
	args := m.Called(ctx, name, email, emailVerified, image, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error) {
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

func (m *MockUserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, user *models.User) (*models.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockUserService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) Create(ctx context.Context, userID, accountID, providerID string, password *string) (*models.Account, error) {
	args := m.Called(ctx, userID, accountID, providerID, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) CreateOAuth2(ctx context.Context, userID, providerAccountID, provider, accessToken string, refreshToken *string, accessTokenExpiresAt, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	args := m.Called(ctx, userID, providerAccountID, provider, accessToken, refreshToken, accessTokenExpiresAt, refreshTokenExpiresAt, scope)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.Account, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) GetByProviderAndAccountID(ctx context.Context, provider, accountID string) (*models.Account, error) {
	args := m.Called(ctx, provider, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	args := m.Called(ctx, userID, fields)
	return args.Error(0)
}

type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) GetByID(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) Create(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error) {
	args := m.Called(ctx, userID, hashedToken, ipAddress, userAgent, maxAge)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) GetByToken(ctx context.Context, hashedToken string) (*models.Session, error) {
	args := m.Called(ctx, hashedToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	args := m.Called(ctx, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionService) DeleteAllExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSessionService) GetDistinctUserIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSessionService) DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error {
	args := m.Called(ctx, userID, maxCount)
	return args.Error(0)
}

type MockVerificationService struct {
	mock.Mock
}

func (m *MockVerificationService) Create(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
	args := m.Called(ctx, userID, hashedToken, vType, value, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationService) GetByToken(ctx context.Context, hashedToken string) (*models.Verification, error) {
	args := m.Called(ctx, hashedToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockVerificationService) DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error {
	args := m.Called(ctx, userID, vType)
	return args.Error(0)
}

func (m *MockVerificationService) IsExpired(verif *models.Verification) bool {
	args := m.Called(verif)
	return args.Bool(0)
}

func (m *MockVerificationService) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) Generate() (string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) Hash(token string) string {
	args := m.Called(token)
	return args.String(0)
}

func (m *MockTokenService) Encrypt(token string) (string, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) Decrypt(encrypted string) (string, error) {
	args := m.Called(encrypted)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

type MockMailerService struct {
	mock.Mock
}

func (m *MockMailerService) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	args := m.Called(ctx, to, subject, text, html)
	return args.Error(0)
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
