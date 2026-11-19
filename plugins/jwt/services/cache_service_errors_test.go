package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

func cacheTestJWKS(t *testing.T) (jwk.Set, string) {
	t.Helper()

	key := testSigningKey(t, "kid-cache", models.SigningKeyAlgorithmRS256)
	pubKey, err := jwk.ParseKey([]byte(key.PublicKey), jwk.WithPEM(true))
	require.NoError(t, err)
	require.NoError(t, pubKey.Set(jwk.KeyIDKey, key.KeyID))

	set := jwk.NewSet()
	require.NoError(t, set.AddKey(pubKey))

	data, err := json.Marshal(set)
	require.NoError(t, err)

	parsed, err := jwk.Parse(data)
	require.NoError(t, err)

	return parsed, string(data)
}

func TestCacheService_CacheJWKS(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	set, _ := cacheTestJWKS(t)

	tests := []struct {
		name    string
		storage func(*testing.T) models.SecondaryStorage
		wantErr string
	}{
		{
			name: "no secondary storage is a no-op",
			storage: func(*testing.T) models.SecondaryStorage {
				return nil
			},
		},
		{
			name: "stores the marshalled set",
			storage: func(*testing.T) models.SecondaryStorage {
				return jwttests.NewInMemoryStorage()
			},
		},
		{
			name: "storage failure is wrapped",
			storage: func(*testing.T) models.SecondaryStorage {
				storage := new(internaltests.MockSecondaryStorage)
				storage.On("Set", ctx, jwksCacheKey, mock.AnythingOfType("string"), mock.Anything).
					Return(assert.AnError).Once()
				return storage
			},
			wantErr: "failed to cache JWKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := tt.storage(t)
			service := &cacheService{
				secondaryStorage: storage,
				logger:           &internaltests.MockLogger{},
				cacheTTL:         time.Minute,
			}

			err := service.CacheJWKS(ctx, set)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if mockStorage, ok := storage.(*internaltests.MockSecondaryStorage); ok {
				mockStorage.AssertExpectations(t)
			}
		})
	}
}

func TestCacheService_InvalidateCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	countedKeys := []models.SigningKey{testSigningKey(t, "kid-counted", models.SigningKeyAlgorithmRS256)}

	tests := []struct {
		name       string
		storage    func(*testing.T) models.SecondaryStorage
		setupKeys  func(*jwttests.MockKeyService)
		wantErr    string
		wantCached bool
	}{
		{
			name: "no secondary storage is a no-op",
			storage: func(*testing.T) models.SecondaryStorage {
				return nil
			},
		},
		{
			name: "delete failure is returned",
			storage: func(*testing.T) models.SecondaryStorage {
				storage := new(internaltests.MockSecondaryStorage)
				storage.On("Delete", ctx, jwksCacheKey).Return(assert.AnError).Once()
				return storage
			},
			wantErr: "failed to delete cache",
		},
		{
			name: "fetch failure is non-fatal",
			storage: func(*testing.T) models.SecondaryStorage {
				return jwttests.NewInMemoryStorage()
			},
			setupKeys: func(keyService *jwttests.MockKeyService) {
				keyService.On("GetPublicKeys", mock.Anything).Return(nil, assert.AnError).Once()
			},
		},
		{
			name: "cache write failure is returned",
			storage: func(*testing.T) models.SecondaryStorage {
				storage := new(internaltests.MockSecondaryStorage)
				storage.On("Delete", ctx, jwksCacheKey).Return(nil).Once()
				storage.On("Set", ctx, jwksCacheKey, mock.AnythingOfType("string"), mock.Anything).
					Return(assert.AnError).Once()
				return storage
			},
			setupKeys: func(keyService *jwttests.MockKeyService) {
				keyService.On("GetPublicKeys", mock.Anything).Return(countedKeys, nil).Once()
			},
			wantErr: "failed to cache JWKS",
		},
		{
			name: "refreshes the cache after deletion",
			storage: func(*testing.T) models.SecondaryStorage {
				return jwttests.NewInMemoryStorage()
			},
			setupKeys: func(keyService *jwttests.MockKeyService) {
				keyService.On("GetPublicKeys", mock.Anything).Return(countedKeys, nil).Once()
			},
			wantCached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := tt.storage(t)
			keyService := new(jwttests.MockKeyService)
			if tt.setupKeys != nil {
				tt.setupKeys(keyService)
			}
			service := &cacheService{
				keyService:       keyService,
				secondaryStorage: storage,
				logger:           &internaltests.MockLogger{},
				cacheTTL:         time.Minute,
			}

			err := service.InvalidateCache(ctx)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tt.wantCached {
				cached, err := storage.Get(ctx, jwksCacheKey)
				require.NoError(t, err)
				require.NotNil(t, cached)
			}
			if mockStorage, ok := storage.(*internaltests.MockSecondaryStorage); ok {
				mockStorage.AssertExpectations(t)
			}
			keyService.AssertExpectations(t)
		})
	}
}

func TestCacheService_GetJWKSWithFallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, encoded := cacheTestJWKS(t)
	keys := []models.SigningKey{testSigningKey(t, "kid-fallback", models.SigningKeyAlgorithmRS256)}

	tests := []struct {
		name       string
		storage    models.SecondaryStorage
		preload    bool
		setupKeys  func(*jwttests.MockKeyService)
		wantErr    string
		wantSet    bool
		wantCached bool
	}{
		{
			name:    "cache hit avoids the database",
			storage: jwttests.NewInMemoryStorage(),
			preload: true,
			wantSet: true,
		},
		{
			name:    "cache miss fetches and caches",
			storage: jwttests.NewInMemoryStorage(),
			setupKeys: func(keyService *jwttests.MockKeyService) {
				keyService.On("GetPublicKeys", mock.Anything).Return(keys, nil).Once()
			},
			wantSet:    true,
			wantCached: true,
		},
		{
			name: "cache miss and database failure",
			storage: func() models.SecondaryStorage {
				return jwttests.NewInMemoryStorage()
			}(),
			setupKeys: func(keyService *jwttests.MockKeyService) {
				keyService.On("GetPublicKeys", mock.Anything).Return(nil, assert.AnError).Once()
			},
			wantErr: "failed to fetch JWKS from database",
		},
		{
			name: "cache write failure still returns the set",
			storage: func() models.SecondaryStorage {
				storage := new(internaltests.MockSecondaryStorage)
				storage.On("Get", ctx, jwksCacheKey).Return(nil, assert.AnError).Once()
				storage.On("Set", ctx, jwksCacheKey, mock.AnythingOfType("string"), mock.Anything).
					Return(assert.AnError).Once()
				return storage
			}(),
			setupKeys: func(keyService *jwttests.MockKeyService) {
				keyService.On("GetPublicKeys", mock.Anything).Return(keys, nil).Once()
			},
			wantSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.preload {
				require.NoError(t, tt.storage.Set(ctx, jwksCacheKey, encoded, nil))
			}
			keyService := new(jwttests.MockKeyService)
			if tt.setupKeys != nil {
				tt.setupKeys(keyService)
			}
			service := &cacheService{
				keyService:       keyService,
				secondaryStorage: tt.storage,
				logger:           &internaltests.MockLogger{},
				cacheTTL:         time.Minute,
			}

			got, err := service.GetJWKSWithFallback(ctx)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				if tt.wantSet {
					assert.Positive(t, got.Len())
				}
			}
			if tt.wantCached {
				cached, err := tt.storage.Get(ctx, jwksCacheKey)
				require.NoError(t, err)
				require.NotNil(t, cached)
			}
			if mockStorage, ok := tt.storage.(*internaltests.MockSecondaryStorage); ok {
				mockStorage.AssertExpectations(t)
			}
			keyService.AssertExpectations(t)
		})
	}
}

func TestNewCacheService(t *testing.T) {
	t.Parallel()

	keyService := new(jwttests.MockKeyService)
	storage := jwttests.NewInMemoryStorage()
	logger := &internaltests.MockLogger{}

	service := NewCacheService(keyService, storage, logger, time.Minute)

	require.NotNil(t, service)
	assert.Equal(t, time.Minute, service.(*cacheService).cacheTTL)
	assert.Equal(t, storage, service.(*cacheService).secondaryStorage)
	assert.Equal(t, logger, service.(*cacheService).logger)
}
