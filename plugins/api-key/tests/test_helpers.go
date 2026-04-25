package tests

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/api-key/repositories"
	"github.com/Authula/authula/plugins/api-key/types"
)

type MockApiKeyRepository struct {
	mock.Mock
}

func (m *MockApiKeyRepository) Create(ctx context.Context, apiKey *types.ApiKey) (*types.ApiKey, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) GetAll(ctx context.Context, ownerType *string, referenceID *string, page int, limit int) ([]*types.ApiKey, int, error) {
	args := m.Called(ctx, ownerType, referenceID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*types.ApiKey), args.Int(1), args.Error(2)
}

func (m *MockApiKeyRepository) GetByID(ctx context.Context, id string) (*types.ApiKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*types.ApiKey, error) {
	args := m.Called(ctx, keyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) Update(ctx context.Context, apiKey *types.ApiKey) (*types.ApiKey, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ApiKey), args.Error(1)
}

func (m *MockApiKeyRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockApiKeyRepository) DeleteExpired(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockApiKeyRepository) DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error {
	return m.Called(ctx, ownerType, referenceID).Error(0)
}

func (m *MockApiKeyRepository) WithTx(tx bun.IDB) repositories.ApiKeyRepository {
	args := m.Called(tx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(repositories.ApiKeyRepository)
}
