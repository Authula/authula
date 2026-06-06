package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestCacheService_GetCachedJWKS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupSvc  func(*cacheTestFixture) CacheService
		setupMock func(*cacheTestFixture)
		wantErr   string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCacheTestFixture(t)
			svc := tt.setupSvc(f)

			_, err := svc.GetCachedJWKS(context.Background())

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
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
		name      string
		noStorage bool
		wantErr   string
	}{
		{
			name:      "no secondary storage",
			noStorage: true,
		},
		{
			name: "success",
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
