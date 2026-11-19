package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
	rootservices "github.com/Authula/authula/services"
)

// MockSigningKeyService covers the signing-key half of the JWT service
// contract. It is embedded by MockJWTService and other plugin-local mocks so
// every fake exposes the same seven-method interface.
type MockSigningKeyService struct{ mock.Mock }

func (m *MockSigningKeyService) GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error) {
	args := m.Called(ctx, purpose, algorithm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SigningKey), args.Error(1)
}

func (m *MockSigningKeyService) GetKeyByID(ctx context.Context, kid string) (*models.SigningKey, error) {
	args := m.Called(ctx, kid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SigningKey), args.Error(1)
}

func (m *MockSigningKeyService) GetPublicKeys(ctx context.Context, purpose string) ([]models.SigningKey, error) {
	args := m.Called(ctx, purpose)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SigningKey), args.Error(1)
}

func (m *MockSigningKeyService) GenerateKey(ctx context.Context, purpose, algorithm string) error {
	args := m.Called(ctx, purpose, algorithm)
	return args.Error(0)
}

func (m *MockSigningKeyService) RotateKey(ctx context.Context, purpose, algorithm string) error {
	args := m.Called(ctx, purpose, algorithm)
	return args.Error(0)
}

func (m *MockSigningKeyService) RotateKeyWithGrace(ctx context.Context, purpose, algorithm string, gracePeriod time.Duration) error {
	args := m.Called(ctx, purpose, algorithm, gracePeriod)
	return args.Error(0)
}

// MockJWTService implements the full public JWT service contract.
type MockJWTService struct {
	MockSigningKeyService
}

func (m *MockJWTService) ValidateToken(ctx context.Context, token string) (*models.Actor, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Actor), args.Error(1)
}

var _ rootservices.JWTService = (*MockJWTService)(nil)
