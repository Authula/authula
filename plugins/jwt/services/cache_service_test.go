package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

func testSigningKey(t *testing.T, kid, algorithm string) models.SigningKey {
	t.Helper()

	var publicKey any
	if algorithm == models.SigningKeyAlgorithmES256 {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		publicKey = &privateKey.PublicKey
	} else {
		privateKey, err := rsa.GenerateKey(rand.Reader, models.DefaultRSAKeySize)
		require.NoError(t, err)
		publicKey = &privateKey.PublicKey
	}

	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})

	return models.SigningKey{
		ID:         "id-" + kid,
		KeyID:      kid,
		Algorithm:  algorithm,
		KeyPurpose: models.SigningKeyPurposeJWTPlugin,
		PublicKey:  string(publicPEM),
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
	}
}

func newCacheTestKeyService(t *testing.T, keys ...models.SigningKey) *jwttests.MockKeyService {
	t.Helper()

	keyService := new(jwttests.MockKeyService)
	keyService.On("GetPublicKeys", mock.Anything).Return(keys, nil)
	return keyService
}

func TestCacheService_FetchJWKSFromDatabase_UsesStoredAlgorithm(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := []models.SigningKey{
		testSigningKey(t, "kid-rs", models.SigningKeyAlgorithmRS256),
		testSigningKey(t, "kid-es", models.SigningKeyAlgorithmES256),
		testSigningKey(t, "kid-ps", models.SigningKeyAlgorithmPS256),
	}

	keyService := newCacheTestKeyService(t, keys...)
	service := new(cacheService)
	service.keyService = keyService
	service.logger = &internaltests.MockLogger{}

	set, err := service.FetchJWKSFromDatabase(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, set.Len())
	keyService.AssertExpectations(t)

	want := map[string]string{
		"kid-rs": models.SigningKeyAlgorithmRS256,
		"kid-es": models.SigningKeyAlgorithmES256,
		"kid-ps": models.SigningKeyAlgorithmPS256,
	}
	for i := range set.Len() {
		key, ok := set.Key(i)
		require.True(t, ok)
		kid, ok := key.KeyID()
		require.True(t, ok)
		algorithm, ok := key.Algorithm()
		require.True(t, ok)
		assert.Equal(t, want[kid], algorithm.String())
	}
}

func TestCacheService_FetchJWKSFromDatabase_SkipsInvalidKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	invalid := models.SigningKey{KeyID: "bad", Algorithm: models.SigningKeyAlgorithmRS256, PublicKey: "not a pem"}
	valid := testSigningKey(t, "good", models.SigningKeyAlgorithmRS256)

	keyService := newCacheTestKeyService(t, invalid, valid)
	service := &cacheService{
		keyService: keyService,
		logger:     &internaltests.MockLogger{},
	}

	set, err := service.FetchJWKSFromDatabase(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, set.Len())
	keyService.AssertExpectations(t)
}

func TestCacheService_FetchJWKSFromDatabase_NoKeys(t *testing.T) {
	t.Parallel()

	keyService := newCacheTestKeyService(t)
	service := &cacheService{
		keyService: keyService,
		logger:     &internaltests.MockLogger{},
	}

	set, err := service.FetchJWKSFromDatabase(context.Background())
	require.ErrorContains(t, err, "no valid keys found")
	assert.Nil(t, set)
	keyService.AssertExpectations(t)
}

func TestCacheService_GetCachedJWKS(t *testing.T) {
	t.Parallel()

	valid := testSigningKey(t, "kid-1", models.SigningKeyAlgorithmRS256)
	pubKey, err := jwk.ParseKey([]byte(valid.PublicKey), jwk.WithPEM(true))
	require.NoError(t, err)
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(pubKey))
	data, err := json.Marshal(set)
	require.NoError(t, err)

	tests := []struct {
		name     string
		storage  models.SecondaryStorage
		wantErr  string
		wantKeys int
	}{
		{
			name:    "no secondary storage",
			storage: nil,
			wantErr: "secondary storage not available",
		},
		{
			name:    "empty cache",
			storage: jwttests.NewInMemoryStorage(),
			wantErr: "cached JWKS is empty or invalid type",
		},
		{
			name:     "cache hit",
			storage:  storageWithValue(string(data)),
			wantKeys: 1,
		},
		{
			name:    "invalid cached value",
			storage: storageWithValue("{not-json"),
			wantErr: "failed to parse cached JWKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &cacheService{
				secondaryStorage: tt.storage,
				logger:           &internaltests.MockLogger{},
			}

			set, err := service.GetCachedJWKS(context.Background())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, set)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantKeys, set.Len())
		})
	}
}

type fixedStorage struct {
	value any
}

func storageWithValue(value any) *fixedStorage { return &fixedStorage{value: value} }

func (s *fixedStorage) Get(context.Context, string) (any, error) { return s.value, nil }
func (s *fixedStorage) Set(context.Context, string, any, *time.Duration) error {
	return nil
}
func (s *fixedStorage) Delete(context.Context, string) error                      { return nil }
func (s *fixedStorage) Incr(context.Context, string, *time.Duration) (int, error) { return 0, nil }
func (s *fixedStorage) TTL(context.Context, string) (*time.Duration, error)       { return nil, nil }
func (s *fixedStorage) Scan(context.Context, string) ([]string, error)            { return nil, nil }
func (s *fixedStorage) Close() error                                              { return nil }
