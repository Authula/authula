package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/repositories"
	"github.com/Authula/authula/plugins/jwt/types"
)

type inMemoryStorage struct {
	mu   sync.RWMutex
	data map[string]string
}

func newInMemoryStorage() *inMemoryStorage {
	return &inMemoryStorage{data: make(map[string]string)}
}

func (s *inMemoryStorage) Get(_ context.Context, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (s *inMemoryStorage) Set(_ context.Context, key string, value any, _ *time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value.(string)
	return nil
}

func (s *inMemoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *inMemoryStorage) Incr(_ context.Context, _ string, _ *time.Duration) (int, error) {
	return 0, nil
}

func (s *inMemoryStorage) TTL(_ context.Context, _ string) (*time.Duration, error) {
	return nil, nil
}

func (s *inMemoryStorage) Scan(_ context.Context, prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []string
	for key := range s.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *inMemoryStorage) Close() error {
	return nil
}

type cacheTestFixture struct {
	db      *bun.DB
	repo    repositories.JWKSRepository
	storage *inMemoryStorage
	logger  models.Logger
	ttl     time.Duration
}

func newCacheTestFixture(t *testing.T) *cacheTestFixture {
	t.Helper()
	db := internaltests.NewSQLiteIntegrationDB(t)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS jwks (
		id TEXT PRIMARY KEY,
		public_key TEXT NOT NULL,
		private_key TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NULL
	)`)
	require.NoError(t, err)

	return &cacheTestFixture{
		db:      db,
		repo:    repositories.NewBunJWKSRepository(db),
		storage: newInMemoryStorage(),
		logger:  &mockLogger{},
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

func TestCacheService_GetJWKSWithFallback_cache_miss(t *testing.T) {
	f := newCacheTestFixture(t)
	ctx := context.Background()

	seedJWKSKeys(t, ctx, f.repo, 1)

	svc := f.newCacheService()

	set, err := svc.GetJWKSWithFallback(ctx)
	require.NoError(t, err)
	require.NotNil(t, set)
	require.Equal(t, 1, set.Len())
}

func TestCacheService_GetJWKSWithFallback_cache_hit(t *testing.T) {
	f := newCacheTestFixture(t)
	ctx := context.Background()

	seedJWKSKeys(t, ctx, f.repo, 1)

	svc := f.newCacheService()

	// First call — cache miss, populates cache from DB
	set1, err := svc.GetJWKSWithFallback(ctx)
	require.NoError(t, err)
	require.NotNil(t, set1)
	require.Equal(t, 1, set1.Len())

	// Second call — should return from cache
	set2, err := svc.GetJWKSWithFallback(ctx)
	require.NoError(t, err)
	require.NotNil(t, set2)
	require.Equal(t, 1, set2.Len())
}

func TestCacheService_InvalidateCache(t *testing.T) {
	f := newCacheTestFixture(t)
	ctx := context.Background()

	seedJWKSKeys(t, ctx, f.repo, 2)

	svc := f.newCacheService()

	// Populate cache
	_, err := svc.GetJWKSWithFallback(ctx)
	require.NoError(t, err)

	// Verify cache is populated
	_, err = svc.GetCachedJWKS(ctx)
	require.NoError(t, err)

	// Invalidate the cache
	err = svc.InvalidateCache(ctx)
	require.NoError(t, err)

	// Cache should be repopulated after InvalidateCache
	set, err := svc.GetCachedJWKS(ctx)
	require.NoError(t, err)
	require.NotNil(t, set)
	require.Equal(t, 2, set.Len())
}

func TestCacheService_GetCachedJWKS_no_storage(t *testing.T) {
	f := newCacheTestFixture(t)
	ctx := context.Background()

	svc := &cacheService{
		repo:             f.repo,
		secondaryStorage: nil,
		logger:           f.logger,
		cacheTTL:         f.ttl,
	}

	_, err := svc.GetCachedJWKS(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "secondary storage not available")
}
