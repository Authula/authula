package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/jwt/migrationset"
	"github.com/Authula/authula/plugins/jwt/types"
)

func setupJWKSRepo(t *testing.T) (*bun.DB, *bunJWKSRepository) {
	t.Helper()
	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)
	err = migrator.Migrate(context.Background(), []migrations.MigrationSet{
		{
			PluginID:   "jwt",
			Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
		},
	})
	require.NoError(t, err)
	return db, &bunJWKSRepository{db: db}
}

func TestBunJWKSRepository(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, repo *bunJWKSRepository, ctx context.Context)
	}{
		{
			name: "StoreJWKSKey",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				key := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "public-key-1",
					PrivateKey: "private-key-1",
				}
				err := repo.StoreJWKSKey(ctx, key)
				require.NoError(t, err)

				keys, err := repo.GetJWKSKeys(ctx)
				require.NoError(t, err)
				require.Len(t, keys, 1)
				require.Equal(t, key.ID, keys[0].ID)
				require.Equal(t, key.PublicKey, keys[0].PublicKey)
				require.Equal(t, key.PrivateKey, keys[0].PrivateKey)
				require.NotZero(t, keys[0].CreatedAt)
				require.Nil(t, keys[0].ExpiresAt)
			},
		},
		{
			name: "GetJWKSKeyByID",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				key := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "public-key-2",
					PrivateKey: "private-key-2",
				}
				err := repo.StoreJWKSKey(ctx, key)
				require.NoError(t, err)

				found, err := repo.GetJWKSKeyByID(ctx, key.ID)
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, key.ID, found.ID)
				require.Equal(t, key.PublicKey, found.PublicKey)
				require.Equal(t, key.PrivateKey, found.PrivateKey)
			},
		},
		{
			name: "GetJWKSKeyByID_not_found",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				found, err := repo.GetJWKSKeyByID(ctx, "non-existent-id")
				require.NoError(t, err)
				require.Nil(t, found)
			},
		},
		{
			name: "GetJWKSKeys_expired_excluded",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				now := time.Now()
				past := now.Add(-1 * time.Hour)

				unexpiredKey := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "public-key-unexpired",
					PrivateKey: "private-key-unexpired",
					ExpiresAt:  nil,
				}
				err := repo.StoreJWKSKey(ctx, unexpiredKey)
				require.NoError(t, err)

				expiredKey := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "public-key-expired",
					PrivateKey: "private-key-expired",
					ExpiresAt:  &past,
				}
				err = repo.StoreJWKSKey(ctx, expiredKey)
				require.NoError(t, err)

				keys, err := repo.GetJWKSKeys(ctx)
				require.NoError(t, err)
				require.Len(t, keys, 1)
				require.Equal(t, unexpiredKey.ID, keys[0].ID)
			},
		},
		{
			name: "MarkKeyExpired",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				key := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "public-key-3",
					PrivateKey: "private-key-3",
				}
				err := repo.StoreJWKSKey(ctx, key)
				require.NoError(t, err)

				past := time.Now().Add(-1 * time.Hour)
				err = repo.MarkKeyExpired(ctx, key.ID, past)
				require.NoError(t, err)

				keys, err := repo.GetJWKSKeys(ctx)
				require.NoError(t, err)
				require.Len(t, keys, 0)
			},
		},
		{
			name: "PurgeExpiredKeys",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				twoDaysAgo := time.Now().Add(-48 * time.Hour)
				key := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "public-key-4",
					PrivateKey: "private-key-4",
					ExpiresAt:  &twoDaysAgo,
				}
				err := repo.StoreJWKSKey(ctx, key)
				require.NoError(t, err)

				err = repo.PurgeExpiredKeys(ctx)
				require.NoError(t, err)

				keys, err := repo.GetJWKSKeys(ctx)
				require.NoError(t, err)
				require.Len(t, keys, 0)
			},
		},
		{
			name: "UpdateJWKSKey",
			run: func(t *testing.T, repo *bunJWKSRepository, ctx context.Context) {
				key := &types.JWKS{
					ID:         uuid.New().String(),
					PublicKey:  "original-public-key",
					PrivateKey: "original-private-key",
				}
				err := repo.StoreJWKSKey(ctx, key)
				require.NoError(t, err)

				key.PublicKey = "updated-public-key"
				err = repo.UpdateJWKSKey(ctx, key)
				require.NoError(t, err)

				found, err := repo.GetJWKSKeyByID(ctx, key.ID)
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, "updated-public-key", found.PublicKey)
				require.Equal(t, "original-private-key", found.PrivateKey)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, repo := setupJWKSRepo(t)
			ctx := context.Background()
			tc.run(t, repo, ctx)
		})
	}
}
