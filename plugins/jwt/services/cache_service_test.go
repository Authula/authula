package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/migrationset"
	"github.com/Authula/authula/plugins/jwt/repositories"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

type cacheTestFixture struct {
	db      *bun.DB
	repo    repositories.JWKSRepository
	storage *jwttests.InMemoryStorage
	logger  models.Logger
	ttl     time.Duration
}

func newCacheTestFixture(t *testing.T) *cacheTestFixture {
	t.Helper()
	db := internaltests.NewSQLiteIntegrationDB(t)

	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)
	err = migrator.Migrate(context.Background(), []migrations.MigrationSet{
		{
			PluginID:   models.PluginJWT.String(),
			Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
		},
	})
	require.NoError(t, err)

	return &cacheTestFixture{
		db:      db,
		repo:    repositories.NewBunJWKSRepository(db),
		storage: jwttests.NewInMemoryStorage(),
		logger:  &internaltests.MockLogger{},
		ttl:     24 * time.Hour,
	}
}

func (f *cacheTestFixture) newCacheService() CacheService {
	return &cacheService{
		repo:             f.repo,
		secondaryStorage: f.storage,
		logger:           f.logger,
		cacheTTL:         f.ttl,
	}
}

func generateTestJWKS(t *testing.T) *types.JWKS {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privBytes, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	pubBytes, _ := x509.MarshalPKIXPublicKey(priv.Public())
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	return &types.JWKS{
		ID:         uuid.New().String(),
		PublicKey:  string(pubPEM),
		PrivateKey: string(privPEM),
		CreatedAt:  time.Now(),
	}
}

func seedJWKSKeys(t *testing.T, ctx context.Context, repo repositories.JWKSRepository, count int) []*types.JWKS {
	t.Helper()
	keys := make([]*types.JWKS, count)
	for i := range count {
		key := generateTestJWKS(t)
		err := repo.StoreJWKSKey(ctx, key)
		require.NoError(t, err)
		keys[i] = key
	}
	return keys
}

// stubSecondaryStorage is a local fake so cache error branches can be
// exercised without touching the shared JWT test mocks.
type stubSecondaryStorage struct {
	value       any
	getErr      error
	setErr      error
	deleteErr   error
	setCalls    int
	deleteCalls int
}

var _ models.SecondaryStorage = (*stubSecondaryStorage)(nil)

func (s *stubSecondaryStorage) Get(context.Context, string) (any, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.value, nil
}

func (s *stubSecondaryStorage) Set(context.Context, string, any, *time.Duration) error {
	s.setCalls++
	return s.setErr
}

func (s *stubSecondaryStorage) Delete(context.Context, string) error {
	s.deleteCalls++
	return s.deleteErr
}

func (s *stubSecondaryStorage) Incr(context.Context, string, *time.Duration) (int, error) {
	return 0, nil
}

func (s *stubSecondaryStorage) TTL(context.Context, string) (*time.Duration, error) {
	return nil, nil
}

func (s *stubSecondaryStorage) Scan(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *stubSecondaryStorage) Close() error { return nil }

func testJWKSetJSON(t *testing.T) string {
	t.Helper()

	pubKey, err := jwk.ParseKey([]byte(generateTestJWKS(t).PublicKey), jwk.WithX509(true))
	require.NoError(t, err)

	set := jwk.NewSet()
	require.NoError(t, set.AddKey(pubKey))

	data, err := json.Marshal(set)
	require.NoError(t, err)

	return string(data)
}

func TestNewCacheService(t *testing.T) {
	t.Parallel()

	f := newCacheTestFixture(t)
	ttl := 12 * time.Hour

	svc := NewCacheService(f.repo, f.storage, f.logger, ttl)

	concrete, ok := svc.(*cacheService)
	require.True(t, ok)
	require.Equal(t, f.repo, concrete.repo)
	require.Equal(t, f.storage, concrete.secondaryStorage)
	require.Equal(t, f.logger, concrete.logger)
	require.Equal(t, ttl, concrete.cacheTTL)
}

func TestCacheService_GetCachedJWKS(t *testing.T) {
	t.Parallel()

	cachedJWKS := testJWKSetJSON(t)

	tests := []struct {
		name     string
		setupSvc func(*cacheTestFixture) CacheService
		wantErr  string
		wantKeys int
	}{
		{
			name: "no secondary storage",
			setupSvc: func(f *cacheTestFixture) CacheService {
				return &cacheService{
					repo:             f.repo,
					secondaryStorage: nil,
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			},
			wantErr: "secondary storage not available",
		},
		{
			name: "empty cache",
			setupSvc: func(f *cacheTestFixture) CacheService {
				return &cacheService{
					repo:             f.repo,
					secondaryStorage: f.storage,
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			},
			wantErr: "cached JWKS is empty or invalid type",
		},
		{
			name: "storage read error",
			setupSvc: func(f *cacheTestFixture) CacheService {
				return &cacheService{
					repo:             f.repo,
					secondaryStorage: &stubSecondaryStorage{getErr: assert.AnError},
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			},
			wantErr: "cache miss",
		},
		{
			name: "cached value has unexpected type",
			setupSvc: func(f *cacheTestFixture) CacheService {
				return &cacheService{
					repo:             f.repo,
					secondaryStorage: &stubSecondaryStorage{value: 42},
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			},
			wantErr: "cached JWKS is empty or invalid type",
		},
		{
			name: "cached value is not a valid JWKS",
			setupSvc: func(f *cacheTestFixture) CacheService {
				return &cacheService{
					repo:             f.repo,
					secondaryStorage: &stubSecondaryStorage{value: "{not-json"},
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			},
			wantErr: "failed to parse cached JWKS",
		},
		{
			name: "cache hit",
			setupSvc: func(f *cacheTestFixture) CacheService {
				return &cacheService{
					repo:             f.repo,
					secondaryStorage: &stubSecondaryStorage{value: cachedJWKS},
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			},
			wantKeys: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			svc := tt.setupSvc(f)

			set, err := svc.GetCachedJWKS(context.Background())

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, set)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, set)
			require.Equal(t, tt.wantKeys, set.Len())
		})
	}
}

func TestCacheService_FetchJWKSFromDatabase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(context.Context, *cacheTestFixture)
		wantKeys int
		wantErr  string
	}{
		{
			name:    "no keys in database",
			setup:   func(ctx context.Context, f *cacheTestFixture) {},
			wantErr: "no valid keys found",
		},
		{
			name: "keys found",
			setup: func(ctx context.Context, f *cacheTestFixture) {
				seedJWKSKeys(t, ctx, f.repo, 2)
			},
			wantKeys: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			svc := f.newCacheService()
			ctx := context.Background()

			tt.setup(ctx, f)

			set, err := svc.FetchJWKSFromDatabase(ctx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, set)
				require.Equal(t, tt.wantKeys, set.Len())
			}
		})
	}
}

func TestCacheService_FetchJWKSFromDatabase_InvalidKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		repoErr      error
		withValidKey bool
		wantKeys     int
		wantErr      string
	}{
		{
			name:    "repository error",
			repoErr: assert.AnError,
			wantErr: "failed to fetch web keys",
		},
		{
			name:    "only invalid public keys",
			wantErr: "no valid keys found",
		},
		{
			name:         "invalid public key is skipped",
			withValidKey: true,
			wantKeys:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			ctx := context.Background()

			repo := f.repo
			var mockRepo *jwttests.MockJWKSRepository
			if tt.repoErr != nil {
				mockRepo = &jwttests.MockJWKSRepository{}
				mockRepo.On("GetJWKSKeys", mock.Anything).Return(nil, tt.repoErr)
				repo = mockRepo
			} else {
				invalid := generateTestJWKS(t)
				invalid.PublicKey = "not a pem"
				require.NoError(t, f.repo.StoreJWKSKey(ctx, invalid))
				if tt.withValidKey {
					seedJWKSKeys(t, ctx, f.repo, 1)
				}
			}

			svc := &cacheService{
				repo:             repo,
				secondaryStorage: f.storage,
				logger:           f.logger,
				cacheTTL:         f.ttl,
			}

			set, err := svc.FetchJWKSFromDatabase(ctx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, set)
			} else {
				require.NoError(t, err)
				require.NotNil(t, set)
				require.Equal(t, tt.wantKeys, set.Len())
			}

			if mockRepo != nil {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestCacheService_GetJWKSWithFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(context.Context, *cacheTestFixture)
		preCache   bool
		wantErr    string
		wantKeyLen int
	}{
		{
			name: "cache miss",
			setup: func(ctx context.Context, f *cacheTestFixture) {
				seedJWKSKeys(t, ctx, f.repo, 1)
			},
			wantKeyLen: 1,
		},
		{
			name: "cache hit",
			setup: func(ctx context.Context, f *cacheTestFixture) {
				seedJWKSKeys(t, ctx, f.repo, 1)
			},
			preCache:   true,
			wantKeyLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			svc := f.newCacheService()
			ctx := context.Background()

			tt.setup(ctx, f)

			// Pre-populate cache if requested
			if tt.preCache {
				_, err := svc.GetJWKSWithFallback(ctx)
				require.NoError(t, err)
			}

			set, err := svc.GetJWKSWithFallback(ctx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, set)
				require.Equal(t, tt.wantKeyLen, set.Len())
			}
		})
	}
}

func TestCacheService_GetJWKSWithFallback_DatabaseFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		storage     *stubSecondaryStorage
		repoErr     error
		seedDB      bool
		wantKeys    int
		wantErr     string
		wantSetCall int
	}{
		{
			name:    "database fetch error",
			storage: &stubSecondaryStorage{},
			repoErr: assert.AnError,
			wantErr: "failed to fetch JWKS from database",
		},
		{
			name:        "cache write error still returns the set",
			storage:     &stubSecondaryStorage{setErr: assert.AnError},
			seedDB:      true,
			wantKeys:    1,
			wantSetCall: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			ctx := context.Background()

			repo := f.repo
			var mockRepo *jwttests.MockJWKSRepository
			if tt.repoErr != nil {
				mockRepo = &jwttests.MockJWKSRepository{}
				mockRepo.On("GetJWKSKeys", mock.Anything).Return(nil, tt.repoErr)
				repo = mockRepo
			}
			if tt.seedDB {
				seedJWKSKeys(t, ctx, f.repo, 1)
			}

			svc := &cacheService{
				repo:             repo,
				secondaryStorage: tt.storage,
				logger:           f.logger,
				cacheTTL:         f.ttl,
			}

			set, err := svc.GetJWKSWithFallback(ctx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, set)
			} else {
				require.NoError(t, err)
				require.NotNil(t, set)
				require.Equal(t, tt.wantKeys, set.Len())
			}
			require.Equal(t, tt.wantSetCall, tt.storage.setCalls)

			if mockRepo != nil {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestCacheService_InvalidateCache_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		storage *stubSecondaryStorage
		repoErr error
		wantErr string
	}{
		{
			name:    "delete error",
			storage: &stubSecondaryStorage{deleteErr: assert.AnError},
			wantErr: "failed to delete cache",
		},
		{
			name:    "database fetch error is ignored",
			storage: &stubSecondaryStorage{},
			repoErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			ctx := context.Background()

			repo := f.repo
			var mockRepo *jwttests.MockJWKSRepository
			if tt.repoErr != nil {
				mockRepo = &jwttests.MockJWKSRepository{}
				mockRepo.On("GetJWKSKeys", mock.Anything).Return(nil, tt.repoErr)
				repo = mockRepo
			}

			svc := &cacheService{
				repo:             repo,
				secondaryStorage: tt.storage,
				logger:           f.logger,
				cacheTTL:         f.ttl,
			}

			err := svc.InvalidateCache(ctx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, tt.storage.deleteCalls)
			require.Zero(t, tt.storage.setCalls)

			if mockRepo != nil {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestCacheService_InvalidateCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		noStorage bool
		setup     func(context.Context, *cacheTestFixture)
		wantErr   string
	}{
		{
			name:      "no secondary storage",
			noStorage: true,
			setup:     func(ctx context.Context, f *cacheTestFixture) {},
		},
		{
			name: "success",
			setup: func(ctx context.Context, f *cacheTestFixture) {
				seedJWKSKeys(t, ctx, f.repo, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)

			var svc CacheService
			if tt.noStorage {
				svc = &cacheService{
					repo:             f.repo,
					secondaryStorage: nil,
					logger:           f.logger,
					cacheTTL:         f.ttl,
				}
			} else {
				svc = f.newCacheService()
			}

			ctx := context.Background()
			tt.setup(ctx, f)

			if tt.noStorage {
				err := svc.InvalidateCache(ctx)
				require.NoError(t, err)
				return
			}

			// Populate cache first
			_, err := svc.GetJWKSWithFallback(ctx)
			require.NoError(t, err)

			err = svc.InvalidateCache(ctx)
			require.NoError(t, err)

			// Cache should be repopulated
			set, err := svc.GetCachedJWKS(ctx)
			require.NoError(t, err)
			require.NotNil(t, set)
		})
	}
}

func TestCacheService_CacheJWKS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		noStorage  bool
		storageErr error
		wantErr    string
	}{
		{
			name:      "no secondary storage",
			noStorage: true,
		},
		{
			name: "success",
		},
		{
			name:       "storage write error",
			storageErr: assert.AnError,
			wantErr:    "failed to cache JWKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)

			var storage models.SecondaryStorage = f.storage
			switch {
			case tt.noStorage:
				storage = nil
			case tt.storageErr != nil:
				storage = &stubSecondaryStorage{setErr: tt.storageErr}
			}

			svc := &cacheService{
				repo:             f.repo,
				secondaryStorage: storage,
				logger:           f.logger,
				cacheTTL:         f.ttl,
			}

			ctx := context.Background()
			seedJWKSKeys(t, ctx, f.repo, 1)
			set, err := svc.FetchJWKSFromDatabase(ctx)
			require.NoError(t, err)

			err = svc.CacheJWKS(ctx, set)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
