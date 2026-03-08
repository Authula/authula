package tests

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/uptrace/bun"
)

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

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	args := m.Called(ctx, user)
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

func (m *MockUserRepository) WithTx(tx bun.IDB) repositories.UserRepository {
	args := m.Called(tx)
	// return self by default unless test stub sets another value
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*MockUserRepository); ok {
			return v
		}
	}
	return m
}

// Account Repository

type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) GetByID(ctx context.Context, id string) (*models.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountRepository) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountRepository) GetAllByUserID(ctx context.Context, userID string) ([]models.Account, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Account), args.Error(1)
}

func (m *MockAccountRepository) GetByUserIDAndProvider(ctx context.Context, userID string, provider string) (*models.Account, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountRepository) GetByProviderAndAccountID(ctx context.Context, provider string, accountID string) (*models.Account, error) {
	args := m.Called(ctx, provider, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountRepository) Create(ctx context.Context, account *models.Account) (*models.Account, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountRepository) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountRepository) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	args := m.Called(ctx, userID, fields)
	return args.Error(0)
}

func (m *MockAccountRepository) WithTx(tx bun.IDB) repositories.AccountRepository {
	args := m.Called(tx)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*MockAccountRepository); ok {
			return v
		}
	}
	return m
}

// Session Repository

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) (*models.Session, error) {
	args := m.Called(ctx, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	args := m.Called(ctx, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error {
	args := m.Called(ctx, userID, maxCount)
	return args.Error(0)
}

func (m *MockSessionRepository) GetDistinctUserIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSessionRepository) WithTx(tx bun.IDB) *MockSessionRepository {
	args := m.Called(tx)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*MockSessionRepository); ok {
			return v
		}
	}
	return m
}

// Verification Repository

type MockVerificationRepository struct {
	mock.Mock
}

func (m *MockVerificationRepository) GetByID(ctx context.Context, id string) (*models.Verification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationRepository) GetByToken(ctx context.Context, token string) (*models.Verification, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationRepository) Create(ctx context.Context, verification *models.Verification) (*models.Verification, error) {
	args := m.Called(ctx, verification)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockVerificationRepository) DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error {
	args := m.Called(ctx, userID, vType)
	return args.Error(0)
}

func (m *MockVerificationRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockVerificationRepository) WithTx(tx bun.IDB) *MockVerificationRepository {
	args := m.Called(tx)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*MockVerificationRepository); ok {
			return v
		}
	}
	return m
}

// Token Repository

type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Generate() (string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockTokenRepository) Hash(token string) string {
	args := m.Called(token)
	return args.String(0)
}

func (m *MockTokenRepository) Encrypt(token string) (string, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockTokenRepository) Decrypt(encrypted string) (string, error) {
	args := m.Called(encrypted)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}
