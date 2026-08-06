package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/repositories"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
)

// Services

type MockAccessControlService struct {
	mock.Mock
}

func (m *MockAccessControlService) RoleExists(ctx context.Context, roleName string) (bool, error) {
	args := m.Called(ctx, roleName)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccessControlService) GetRolePermissionsByName(ctx context.Context, roleName string) ([]string, error) {
	args := m.Called(ctx, roleName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAccessControlService) GetRoleWeightByName(ctx context.Context, roleName string) (int, error) {
	args := m.Called(ctx, roleName)
	return args.Int(0), args.Error(1)
}

func (m *MockAccessControlService) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	return m.Called(ctx, permissionKeys).Error(0)
}

func (m *MockAccessControlService) EnsurePermissions(ctx context.Context, permissions []rootservices.PermissionDefinition) error {
	return m.Called(ctx, permissions).Error(0)
}

type MockApiKeyService struct {
	mock.Mock
}

func (m *MockApiKeyService) Create(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	args := m.Called(ctx, actor, req)
	if resp, ok := args.Get(0).(*types.CreateApiKeyResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) GetByID(ctx context.Context, actor *models.Actor, id string) (*types.ApiKey, error) {
	args := m.Called(ctx, actor, id)
	if resp, ok := args.Get(0).(*types.ApiKey); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) GetAll(ctx context.Context, actor *models.Actor, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	args := m.Called(ctx, actor, req)
	if resp, ok := args.Get(0).(*types.GetAllApiKeysResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) Update(ctx context.Context, actor *models.Actor, id string, req types.UpdateApiKeyData) (*types.ApiKey, error) {
	args := m.Called(ctx, actor, id, req)
	if resp, ok := args.Get(0).(*types.ApiKey); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) Delete(ctx context.Context, actor *models.Actor, id string) error {
	return m.Called(ctx, actor, id).Error(0)
}

func (m *MockApiKeyService) RecordLastRequest(ctx context.Context, id string, timestamp time.Time) (*types.ApiKey, error) {
	args := m.Called(ctx, id, timestamp)
	if resp, ok := args.Get(0).(*types.ApiKey); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) DeleteExpired(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockApiKeyService) DeleteAllByOwner(ctx context.Context, actor *models.Actor, ownerType string, ownerID string) error {
	return m.Called(ctx, actor, ownerType, ownerID).Error(0)
}

func (m *MockApiKeyService) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*types.VerifyApiKeyResult); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockApiKeyService) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	return m.Called(ctx, permissionKeys).Error(0)
}

// Repositories

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

func (m *MockApiKeyRepository) GetAll(ctx context.Context, ownerType *string, ownerID *string, page int, limit int) ([]*types.ApiKey, int, error) {
	args := m.Called(ctx, ownerType, ownerID, page, limit)
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

func (m *MockApiKeyRepository) DeleteAllByOwner(ctx context.Context, ownerType string, ownerID string) error {
	return m.Called(ctx, ownerType, ownerID).Error(0)
}

func (m *MockApiKeyRepository) WithTx(tx bun.IDB) repositories.ApiKeyRepository {
	args := m.Called(tx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(repositories.ApiKeyRepository)
}
