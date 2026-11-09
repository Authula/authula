package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	coreservices "github.com/Authula/authula/services"

	"github.com/Authula/authula/plugins/jwt/repositories"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

const testEncryptedPrivateKey = "encrypted-private-key"

type keyServiceTestCapture struct {
	storedKeys []*types.JWKS
	expiredIDs []string
	expiredAt  map[string]time.Time
}

func newKeyServiceTestCapture() *keyServiceTestCapture {
	return &keyServiceTestCapture{expiredAt: make(map[string]time.Time)}
}

func newKeyServiceTestDeps() (KeyService, *jwttests.MockJWKSRepository, *jwttests.MockTokenServiceCore) {
	repo := new(jwttests.MockJWKSRepository)
	tokenSvc := new(jwttests.MockTokenServiceCore)
	svc := NewKeyService(repo, &internaltests.MockLogger{}, tokenSvc)
	return svc, repo, tokenSvc
}

func expectEncryptPrivateKey(tokenSvc *jwttests.MockTokenServiceCore, encrypted string, err error) {
	tokenSvc.On("Encrypt", mock.MatchedBy(func(value string) bool {
		return strings.Contains(value, "BEGIN PRIVATE KEY")
	})).Return(encrypted, err).Once()
}

func captureStoredKey(repo *jwttests.MockJWKSRepository, capture *keyServiceTestCapture, err error) {
	repo.On("StoreJWKSKey", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capture.storedKeys = append(capture.storedKeys, args.Get(1).(*types.JWKS))
		}).
		Return(err).
		Once()
}

func captureExpiredKey(repo *jwttests.MockJWKSRepository, capture *keyServiceTestCapture, err error) {
	repo.On("MarkKeyExpired", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			id := args.String(1)
			capture.expiredIDs = append(capture.expiredIDs, id)
			capture.expiredAt[id] = args.Get(2).(time.Time)
		}).
		Return(err).
		Once()
}

func assertValidStoredKey(t *testing.T, key *types.JWKS, wantPrivateKey string) {
	t.Helper()

	require.NotNil(t, key)
	_, err := uuid.Parse(key.ID)
	require.NoError(t, err)

	block, _ := pem.Decode([]byte(key.PublicKey))
	require.NotNil(t, block)
	assert.Equal(t, "PUBLIC KEY", block.Type)

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	assert.IsType(t, ed25519.PublicKey{}, parsed)

	assert.Equal(t, wantPrivateKey, key.PrivateKey)
	assert.Nil(t, key.ExpiresAt)
	assert.WithinDuration(t, time.Now(), key.CreatedAt, time.Minute)
}

func TestNewKeyService(t *testing.T) {
	t.Parallel()

	repo := new(jwttests.MockJWKSRepository)
	tokenSvc := new(jwttests.MockTokenServiceCore)
	logger := &internaltests.MockLogger{}

	tests := []struct {
		name       string
		repo       repositories.JWKSRepository
		logger     models.Logger
		tokenSvc   coreservices.TokenService
		wantNilDep bool
	}{
		{
			name:     "wires all dependencies",
			repo:     repo,
			logger:   logger,
			tokenSvc: tokenSvc,
		},
		{
			name:       "accepts nil dependencies",
			wantNilDep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewKeyService(tt.repo, tt.logger, tt.tokenSvc)

			ks, ok := svc.(*keyService)
			require.True(t, ok)
			require.NotNil(t, ks)

			if tt.wantNilDep {
				assert.Nil(t, ks.repo)
				assert.Nil(t, ks.logger)
				assert.Nil(t, ks.tokenService)
				return
			}

			assert.Equal(t, tt.repo, ks.repo)
			assert.Equal(t, tt.logger, ks.logger)
			assert.Equal(t, tt.tokenSvc, ks.tokenService)
		})
	}
}

func TestKeyService_GenerateKeysIfMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setup            func(*jwttests.MockJWKSRepository, *jwttests.MockTokenServiceCore, *keyServiceTestCapture)
		wantErr          string
		wantStoreAttempt bool
		wantStoredKey    bool
	}{
		{
			name: "repository error",
			setup: func(repo *jwttests.MockJWKSRepository, _ *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(nil, assert.AnError).Once()
			},
			wantErr: "failed to get keys",
		},
		{
			name: "generates key when none exist",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return([]*types.JWKS{}, nil).Once()
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, nil)
			},
			wantStoredKey: true,
		},
		{
			name: "skips generation when keys exist",
			setup: func(repo *jwttests.MockJWKSRepository, _ *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return([]*types.JWKS{
					{
						ID:         "pre-seeded-key",
						PublicKey:  "pre-seeded-public-key",
						PrivateKey: "pre-seeded-private-key",
						CreatedAt:  time.Now(),
					},
				}, nil).Once()
			},
		},
		{
			name: "encrypt private key error",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return([]*types.JWKS{}, nil).Once()
				expectEncryptPrivateKey(tokenSvc, "", assert.AnError)
			},
			wantErr: "failed to encrypt private key",
		},
		{
			name: "store key error",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return([]*types.JWKS{}, nil).Once()
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, assert.AnError)
			},
			wantErr:          "failed to store key",
			wantStoreAttempt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenSvc := newKeyServiceTestDeps()
			capture := newKeyServiceTestCapture()
			tt.setup(repo, tokenSvc, capture)

			err := svc.GenerateKeysIfMissing(context.Background())

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantStoredKey {
				require.Len(t, capture.storedKeys, 1)
				assertValidStoredKey(t, capture.storedKeys[0], testEncryptedPrivateKey)
			} else if tt.wantStoreAttempt {
				require.Len(t, capture.storedKeys, 1)
			} else {
				assert.Empty(t, capture.storedKeys)
			}

			repo.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestKeyService_GetActiveKey(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		keys    []*types.JWKS
		repoErr error
		wantID  string
		wantErr string
	}{
		{
			name:    "repository error",
			repoErr: assert.AnError,
			wantErr: "failed to get keys",
		},
		{
			name:    "no keys",
			keys:    []*types.JWKS{},
			wantErr: "no active key found",
		},
		{
			name: "single key",
			keys: []*types.JWKS{
				{ID: "single-key", CreatedAt: now},
			},
			wantID: "single-key",
		},
		{
			name: "returns most recent key",
			keys: []*types.JWKS{
				{ID: "oldest-key", CreatedAt: now.Add(-2 * time.Hour)},
				{ID: "newest-key", CreatedAt: now.Add(-1 * time.Hour)},
				{ID: "middle-key", CreatedAt: now.Add(-90 * time.Minute)},
			},
			wantID: "newest-key",
		},
		{
			name: "keeps first key when creation times are equal",
			keys: []*types.JWKS{
				{ID: "first-key", CreatedAt: now},
				{ID: "second-key", CreatedAt: now},
			},
			wantID: "first-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenSvc := newKeyServiceTestDeps()
			repo.On("GetJWKSKeys", mock.Anything).Return(tt.keys, tt.repoErr).Once()

			active, err := svc.GetActiveKey(context.Background())

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, active)
			} else {
				require.NoError(t, err)
				require.NotNil(t, active)
				assert.Equal(t, tt.wantID, active.ID)
			}

			repo.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestKeyService_IsKeyRotationDue(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name             string
		keys             []*types.JWKS
		repoErr          error
		rotationInterval time.Duration
		want             bool
	}{
		{
			name:             "repository error returns false",
			repoErr:          assert.AnError,
			rotationInterval: time.Hour,
			want:             false,
		},
		{
			name:             "no keys returns false",
			keys:             []*types.JWKS{},
			rotationInterval: time.Hour,
			want:             false,
		},
		{
			name: "key older than interval is due",
			keys: []*types.JWKS{
				{ID: "old-key", CreatedAt: now.Add(-2 * time.Hour)},
			},
			rotationInterval: time.Hour,
			want:             true,
		},
		{
			name: "newest key older than interval is due",
			keys: []*types.JWKS{
				{ID: "older-key", CreatedAt: now.Add(-3 * time.Hour)},
				{ID: "old-key", CreatedAt: now.Add(-2 * time.Hour)},
			},
			rotationInterval: time.Hour,
			want:             true,
		},
		{
			name: "key newer than interval is not due",
			keys: []*types.JWKS{
				{ID: "recent-key", CreatedAt: now.Add(-30 * time.Minute)},
			},
			rotationInterval: time.Hour,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenSvc := newKeyServiceTestDeps()
			repo.On("GetJWKSKeys", mock.Anything).Return(tt.keys, tt.repoErr).Once()

			due := svc.IsKeyRotationDue(context.Background(), tt.rotationInterval)

			assert.Equal(t, tt.want, due)
			repo.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestKeyService_RotateKeysIfNeeded(t *testing.T) {
	t.Parallel()

	const (
		rotationInterval = 24 * time.Hour
		gracePeriod      = time.Hour
	)

	now := time.Now()
	dueKeys := []*types.JWKS{
		{
			ID:         "old-key",
			PublicKey:  "old-public-key",
			PrivateKey: "old-private-key",
			CreatedAt:  now.Add(-25 * time.Hour),
		},
	}
	multipleDueKeys := []*types.JWKS{
		{
			ID:         "old-key",
			PublicKey:  "old-public-key",
			PrivateKey: "old-private-key",
			CreatedAt:  now.Add(-26 * time.Hour),
		},
		{
			ID:         "older-key",
			PublicKey:  "older-public-key",
			PrivateKey: "older-private-key",
			CreatedAt:  now.Add(-48 * time.Hour),
		},
	}
	recentKeys := []*types.JWKS{
		{
			ID:        "recent-key",
			CreatedAt: now.Add(-time.Hour),
		},
	}

	tests := []struct {
		name               string
		setup              func(*jwttests.MockJWKSRepository, *jwttests.MockTokenServiceCore, *keyServiceTestCapture)
		useNilInvalidator  bool
		cacheErr           error
		wantRotated        bool
		wantErr            string
		wantExpiredIDs     []string
		wantStoreAttempt   bool
		wantStoredKey      bool
		wantCacheCallCount int
	}{
		{
			name: "rotation not due",
			setup: func(repo *jwttests.MockJWKSRepository, _ *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(recentKeys, nil).Once()
			},
			wantRotated: false,
		},
		{
			name: "initial repository error is swallowed",
			setup: func(repo *jwttests.MockJWKSRepository, _ *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(nil, assert.AnError).Once()
			},
			wantRotated: false,
		},
		{
			name: "rotates due key with nil invalidator",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(dueKeys, nil).Times(2)
				captureExpiredKey(repo, capture, nil)
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, nil)
			},
			useNilInvalidator: true,
			wantRotated:       true,
			wantExpiredIDs:    []string{"old-key"},
			wantStoredKey:     true,
		},
		{
			name: "rotates all due keys",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(multipleDueKeys, nil).Times(2)
				captureExpiredKey(repo, capture, nil)
				captureExpiredKey(repo, capture, nil)
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, nil)
			},
			useNilInvalidator: true,
			wantRotated:       true,
			wantExpiredIDs:    []string{"old-key", "older-key"},
			wantStoredKey:     true,
		},
		{
			name: "rotation fetch error",
			setup: func(repo *jwttests.MockJWKSRepository, _ *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(dueKeys, nil).Once()
				repo.On("GetJWKSKeys", mock.Anything).Return(nil, assert.AnError).Once()
			},
			wantErr:        "failed to get keys for rotation",
			wantRotated:    false,
			wantExpiredIDs: nil,
		},
		{
			name: "mark old key expired error",
			setup: func(repo *jwttests.MockJWKSRepository, _ *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(dueKeys, nil).Times(2)
				captureExpiredKey(repo, capture, assert.AnError)
			},
			wantErr:        "failed to set expiration on old key",
			wantRotated:    false,
			wantExpiredIDs: []string{"old-key"},
		},
		{
			name: "store new key error",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(dueKeys, nil).Times(2)
				captureExpiredKey(repo, capture, nil)
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, assert.AnError)
			},
			wantErr:          "failed to generate new key",
			wantRotated:      false,
			wantExpiredIDs:   []string{"old-key"},
			wantStoreAttempt: true,
		},
		{
			name: "invalidates cache after rotation",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(dueKeys, nil).Times(2)
				captureExpiredKey(repo, capture, nil)
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, nil)
			},
			cacheErr:           nil,
			wantRotated:        true,
			wantExpiredIDs:     []string{"old-key"},
			wantStoredKey:      true,
			wantCacheCallCount: 1,
		},
		{
			name: "cache invalidation error does not fail rotation",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				repo.On("GetJWKSKeys", mock.Anything).Return(dueKeys, nil).Times(2)
				captureExpiredKey(repo, capture, nil)
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, nil)
			},
			cacheErr:           assert.AnError,
			wantRotated:        true,
			wantExpiredIDs:     []string{"old-key"},
			wantStoredKey:      true,
			wantCacheCallCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenSvc := newKeyServiceTestDeps()
			capture := newKeyServiceTestCapture()
			tt.setup(repo, tokenSvc, capture)

			cacheCalls := 0
			var invalidateCacheFunc func(context.Context) error
			if !tt.useNilInvalidator {
				invalidateCacheFunc = func(context.Context) error {
					cacheCalls++
					return tt.cacheErr
				}
			}

			rotated, err := svc.RotateKeysIfNeeded(
				context.Background(),
				rotationInterval,
				gracePeriod,
				invalidateCacheFunc,
			)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRotated, rotated)
			assert.Equal(t, tt.wantCacheCallCount, cacheCalls)

			if tt.wantExpiredIDs != nil {
				assert.ElementsMatch(t, tt.wantExpiredIDs, capture.expiredIDs)
				for _, expiredAt := range capture.expiredAt {
					assert.WithinDuration(t, time.Now().Add(gracePeriod), expiredAt, 2*time.Second)
				}
			} else {
				assert.Empty(t, capture.expiredIDs)
			}

			if tt.wantStoredKey {
				require.Len(t, capture.storedKeys, 1)
				assertValidStoredKey(t, capture.storedKeys[0], testEncryptedPrivateKey)
			} else if tt.wantStoreAttempt {
				require.Len(t, capture.storedKeys, 1)
			} else {
				assert.Empty(t, capture.storedKeys)
			}

			repo.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestGenerateKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		validate func(*testing.T, any, any)
	}{
		{
			name: "returns an ed25519 private key",
			validate: func(t *testing.T, priv any, _ any) {
				t.Helper()
				key, ok := priv.(ed25519.PrivateKey)
				require.True(t, ok)
				assert.Len(t, key, ed25519.PrivateKeySize)
				assert.Len(t, key.Seed(), ed25519.SeedSize)
			},
		},
		{
			name: "returns an ed25519 public key",
			validate: func(t *testing.T, _ any, pub any) {
				t.Helper()
				key, ok := pub.(ed25519.PublicKey)
				require.True(t, ok)
				assert.Len(t, key, ed25519.PublicKeySize)
			},
		},
		{
			name: "public key matches private key",
			validate: func(t *testing.T, priv any, pub any) {
				t.Helper()
				privateKey, ok := priv.(ed25519.PrivateKey)
				require.True(t, ok)
				publicKey, ok := pub.(ed25519.PublicKey)
				require.True(t, ok)
				assert.True(t, publicKey.Equal(privateKey.Public()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			priv, pub, err := generateKey()
			require.NoError(t, err)

			tt.validate(t, priv, pub)
		})
	}

	t.Run("generates distinct keys per call", func(t *testing.T) {
		t.Parallel()

		_, firstPub, err := generateKey()
		require.NoError(t, err)
		_, secondPub, err := generateKey()
		require.NoError(t, err)

		first, ok := firstPub.(ed25519.PublicKey)
		require.True(t, ok)
		second, ok := secondPub.(ed25519.PublicKey)
		require.True(t, ok)

		assert.False(t, first.Equal(second))
	})
}

func TestKeyService_GenerateAndStoreKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setup            func(*jwttests.MockJWKSRepository, *jwttests.MockTokenServiceCore, *keyServiceTestCapture)
		wantErr          string
		wantStoreAttempt bool
		wantStoredKey    bool
	}{
		{
			name: "stores generated key",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, nil)
			},
			wantStoredKey: true,
		},
		{
			name: "encrypt private key error",
			setup: func(_ *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, _ *keyServiceTestCapture) {
				expectEncryptPrivateKey(tokenSvc, "", assert.AnError)
			},
			wantErr: "failed to encrypt private key",
		},
		{
			name: "store key error",
			setup: func(repo *jwttests.MockJWKSRepository, tokenSvc *jwttests.MockTokenServiceCore, capture *keyServiceTestCapture) {
				expectEncryptPrivateKey(tokenSvc, testEncryptedPrivateKey, nil)
				captureStoredKey(repo, capture, assert.AnError)
			},
			wantErr:          "failed to store key",
			wantStoreAttempt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo, tokenSvc := newKeyServiceTestDeps()
			ks, ok := svc.(*keyService)
			require.True(t, ok)

			capture := newKeyServiceTestCapture()
			tt.setup(repo, tokenSvc, capture)

			err := ks.generateAndStoreKey(context.Background())

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantStoredKey {
				require.Len(t, capture.storedKeys, 1)
				assertValidStoredKey(t, capture.storedKeys[0], testEncryptedPrivateKey)
			} else if tt.wantStoreAttempt {
				require.Len(t, capture.storedKeys, 1)
			} else {
				assert.Empty(t, capture.storedKeys)
			}

			repo.AssertExpectations(t)
			tokenSvc.AssertExpectations(t)
		})
	}
}

func TestPrivateKeyToPEM(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    any
		wantErr  string
		validate func(*testing.T, *pem.Block)
	}{
		{
			name:  "valid ed25519 private key",
			input: privateKey,
			validate: func(t *testing.T, block *pem.Block) {
				t.Helper()
				assert.Equal(t, "PRIVATE KEY", block.Type)

				parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
				require.NoError(t, err)
				parsedKey, ok := parsed.(ed25519.PrivateKey)
				require.True(t, ok)
				assert.True(t, parsedKey.Equal(privateKey))
			},
		},
		{
			name:    "nil key",
			input:   nil,
			wantErr: "unsupported private key type",
		},
		{
			name:    "string is unsupported",
			input:   "not-a-key",
			wantErr: "unsupported private key type",
		},
		{
			name:    "rsa private key is unsupported",
			input:   new(rsa.PrivateKey),
			wantErr: "unsupported private key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := privateKeyToPEM(tt.input)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, encoded)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			block, _ := pem.Decode(encoded)
			require.NotNil(t, block)
			tt.validate(t, block)
		})
	}
}

func TestPublicKeyToPEM(t *testing.T) {
	t.Parallel()

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    any
		wantErr  string
		validate func(*testing.T, *pem.Block)
	}{
		{
			name:  "valid ed25519 public key",
			input: publicKey,
			validate: func(t *testing.T, block *pem.Block) {
				t.Helper()
				assert.Equal(t, "PUBLIC KEY", block.Type)

				parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
				require.NoError(t, err)
				parsedKey, ok := parsed.(ed25519.PublicKey)
				require.True(t, ok)
				assert.True(t, parsedKey.Equal(publicKey))
			},
		},
		{
			name:    "nil key",
			input:   nil,
			wantErr: "unsupported public key type",
		},
		{
			name:    "string is unsupported",
			input:   "not-a-key",
			wantErr: "unsupported public key type",
		},
		{
			name:    "rsa public key is unsupported",
			input:   new(rsa.PublicKey),
			wantErr: "unsupported public key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := publicKeyToPEM(tt.input)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, encoded)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			block, _ := pem.Decode(encoded)
			require.NotNil(t, block)
			tt.validate(t, block)
		})
	}
}
