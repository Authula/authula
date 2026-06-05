package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreservices "github.com/Authula/authula/services"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/jwt/repositories"
	"github.com/Authula/authula/plugins/jwt/types"
)

type nopTokenService struct{}

func (nopTokenService) Generate() (string, error)                { return "", nil }
func (nopTokenService) Hash(token string) string                 { return token }
func (nopTokenService) Encrypt(token string) (string, error)     { return token, nil }
func (nopTokenService) Decrypt(encrypted string) (string, error) { return encrypted, nil }

func setupKeyServiceTest(t *testing.T) (KeyService, repositories.JWKSRepository) {
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

	repo := repositories.NewBunJWKSRepository(db)
	logger := &internaltests.MockLogger{}
	coreTokenSvc := nopTokenService{}

	svc := NewKeyService(repo, logger, coreservices.TokenService(coreTokenSvc), "test-secret")
	return svc, repo
}

func TestKeyService(t *testing.T) {
	t.Run("GenerateKeysIfMissing_no_keys", func(t *testing.T) {
		svc, repo := setupKeyServiceTest(t)
		ctx := context.Background()

		keys, err := repo.GetJWKSKeys(ctx)
		require.NoError(t, err)
		require.Len(t, keys, 0)

		err = svc.GenerateKeysIfMissing(ctx)
		require.NoError(t, err)

		keys, err = repo.GetJWKSKeys(ctx)
		require.NoError(t, err)
		require.Len(t, keys, 1)
		require.NotEmpty(t, keys[0].ID)
		assert.Contains(t, keys[0].PublicKey, "BEGIN PUBLIC KEY")
		assert.Contains(t, keys[0].PrivateKey, "BEGIN PRIVATE KEY")
	})

	t.Run("GenerateKeysIfMissing_keys_exist", func(t *testing.T) {
		svc, repo := setupKeyServiceTest(t)
		ctx := context.Background()

		preSeedKey := &types.JWKS{
			ID:         "pre-seeded-key",
			PublicKey:  "pre-seeded-public-key",
			PrivateKey: "pre-seeded-private-key",
		}
		err := repo.StoreJWKSKey(ctx, preSeedKey)
		require.NoError(t, err)

		err = svc.GenerateKeysIfMissing(ctx)
		require.NoError(t, err)

		keys, err := repo.GetJWKSKeys(ctx)
		require.NoError(t, err)
		require.Len(t, keys, 1)
		require.Equal(t, "pre-seeded-key", keys[0].ID)
	})

	t.Run("GetActiveKey", func(t *testing.T) {
		svc, repo := setupKeyServiceTest(t)
		ctx := context.Background()

		now := time.Now()
		key1 := &types.JWKS{
			ID:         "key-1",
			PublicKey:  "public-key-1",
			PrivateKey: "private-key-1",
			CreatedAt:  now.Add(-1 * time.Hour),
		}
		err := repo.StoreJWKSKey(ctx, key1)
		require.NoError(t, err)

		active, err := svc.GetActiveKey(ctx)
		require.NoError(t, err)
		require.NotNil(t, active)
		require.Equal(t, "key-1", active.ID)

		key2 := &types.JWKS{
			ID:         "key-2",
			PublicKey:  "public-key-2",
			PrivateKey: "private-key-2",
			CreatedAt:  now,
		}
		err = repo.StoreJWKSKey(ctx, key2)
		require.NoError(t, err)

		active, err = svc.GetActiveKey(ctx)
		require.NoError(t, err)
		require.NotNil(t, active)
		require.Equal(t, "key-2", active.ID)
	})

	t.Run("RotateKeysIfNeeded_due", func(t *testing.T) {
		svc, repo := setupKeyServiceTest(t)
		ctx := context.Background()

		oldKey := &types.JWKS{
			ID:         "old-key",
			PublicKey:  "old-public-key",
			PrivateKey: "old-private-key",
			CreatedAt:  time.Now().Add(-25 * time.Hour),
		}
		err := repo.StoreJWKSKey(ctx, oldKey)
		require.NoError(t, err)

		rotated, err := svc.RotateKeysIfNeeded(ctx, 24*time.Hour, 1*time.Hour, nil)
		require.NoError(t, err)
		require.True(t, rotated)

		storedOldKey, err := repo.GetJWKSKeyByID(ctx, "old-key")
		require.NoError(t, err)
		require.NotNil(t, storedOldKey)
		require.NotNil(t, storedOldKey.ExpiresAt)

		keys, err := repo.GetJWKSKeys(ctx)
		require.NoError(t, err)
		require.Len(t, keys, 2)
	})

	t.Run("RotateKeysIfNeeded_not_due", func(t *testing.T) {
		svc, repo := setupKeyServiceTest(t)
		ctx := context.Background()

		key := &types.JWKS{
			ID:         "recent-key",
			PublicKey:  "recent-public-key",
			PrivateKey: "recent-private-key",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		}
		err := repo.StoreJWKSKey(ctx, key)
		require.NoError(t, err)

		rotated, err := svc.RotateKeysIfNeeded(ctx, 24*time.Hour, 1*time.Hour, nil)
		require.NoError(t, err)
		require.False(t, rotated)

		keys, err := repo.GetJWKSKeys(ctx)
		require.NoError(t, err)
		require.Len(t, keys, 1)
	})
}
