package tests

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/plugins/api-key/types"
)

type MockApiKeyService struct {
	mock.Mock
}

func (m *MockApiKeyService) Create(ctx context.Context, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*types.CreateApiKeyResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) GetByID(ctx context.Context, id string) (*types.ApiKey, error) {
	args := m.Called(ctx, id)
	if resp, ok := args.Get(0).(*types.ApiKey); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) GetAll(ctx context.Context, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*types.GetAllApiKeysResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) Update(ctx context.Context, id string, req types.UpdateApiKeyRequest) (*types.ApiKey, error) {
	args := m.Called(ctx, id, req)
	if resp, ok := args.Get(0).(*types.ApiKey); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockApiKeyService) DeleteExpired(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockApiKeyService) DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error {
	return m.Called(ctx, ownerType, referenceID).Error(0)
}

func (m *MockApiKeyService) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*types.VerifyApiKeyResult); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}
