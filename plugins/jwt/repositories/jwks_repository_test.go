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

func newJWKSRecord() *types.JWKS {
	return &types.JWKS{
		ID:         uuid.New().String(),
		PublicKey:  uuid.New().String(),
		PrivateKey: uuid.New().String(),
	}
}

func TestNewBunJWKSRepository(t *testing.T) {
	t.Parallel()

	db, _ := setupJWKSRepo(t)
	repo := NewBunJWKSRepository(db)
	require.NotNil(t, repo)

	ctx := context.Background()
	key := newJWKSRecord()

	require.NoError(t, repo.StoreJWKSKey(ctx, key))

	found, err := repo.GetJWKSKeys(ctx)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, key.ID, found[0].ID)
}

func TestBunJWKSRepository_DatabaseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		call       func(ctx context.Context, repo *bunJWKSRepository) error
		wantErrMsg string
	}{
		{
			name: "GetJWKSKeys wraps the select failure",
			call: func(ctx context.Context, repo *bunJWKSRepository) error {
				_, err := repo.GetJWKSKeys(ctx)
				return err
			},
			wantErrMsg: "failed to get jwks keys",
		},
		{
			name: "StoreJWKSKey wraps the insert failure",
			call: func(ctx context.Context, repo *bunJWKSRepository) error {
				return repo.StoreJWKSKey(ctx, newJWKSRecord())
			},
			wantErrMsg: "failed to store jwks key",
		},
		{
			name: "MarkKeyExpired wraps the update failure",
			call: func(ctx context.Context, repo *bunJWKSRepository) error {
				return repo.MarkKeyExpired(ctx, "any-id", time.Now())
			},
			wantErrMsg: "failed to mark key as expired",
		},
		{
			name: "PurgeExpiredKeys wraps the delete failure",
			call: func(ctx context.Context, repo *bunJWKSRepository) error {
				return repo.PurgeExpiredKeys(ctx)
			},
			wantErrMsg: "failed to purge expired keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, repo := setupJWKSRepo(t)
			require.NoError(t, db.Close())

			err := tt.call(context.Background(), repo)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestBunJWKSRepository_StoreDuplicate(t *testing.T) {
	t.Parallel()

	_, repo := setupJWKSRepo(t)
	ctx := context.Background()

	key := newJWKSRecord()
	require.NoError(t, repo.StoreJWKSKey(ctx, key))

	duplicate := newJWKSRecord()
	duplicate.ID = key.ID

	err := repo.StoreJWKSKey(ctx, duplicate)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to store jwks key")
}

func TestBunJWKSRepository_MissingKeyUpdates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(ctx context.Context, repo *bunJWKSRepository) error
	}{
		{
			name: "marking a missing key expired is a no-op",
			call: func(ctx context.Context, repo *bunJWKSRepository) error {
				return repo.MarkKeyExpired(ctx, "missing-key", time.Now().Add(-time.Hour))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupJWKSRepo(t)
			ctx := context.Background()

			require.NoError(t, tt.call(ctx, repo))

			keys, err := repo.GetJWKSKeys(ctx)
			require.NoError(t, err)
			require.Empty(t, keys)
		})
	}
}

func TestBunJWKSRepository_PurgeExpiredKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt *time.Time
		wantPurge bool
	}{
		{
			name:      "purges keys expired more than a day ago",
			expiresAt: new(time.Now().Add(-48 * time.Hour)),
			wantPurge: true,
		},
		{
			name:      "keeps keys expired less than a day ago",
			expiresAt: new(time.Now().Add(-time.Hour)),
			wantPurge: false,
		},
		{
			name:      "keeps keys that expire in the future",
			expiresAt: new(time.Now().Add(time.Hour)),
			wantPurge: false,
		},
		{
			name:      "keeps keys without an expiry",
			expiresAt: nil,
			wantPurge: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, repo := setupJWKSRepo(t)
			ctx := context.Background()

			key := newJWKSRecord()
			key.ExpiresAt = tt.expiresAt
			require.NoError(t, repo.StoreJWKSKey(ctx, key))

			require.NoError(t, repo.PurgeExpiredKeys(ctx))

			exists, err := db.NewSelect().Model((*types.JWKS)(nil)).Where("id = ?", key.ID).Exists(ctx)
			require.NoError(t, err)
			require.Equal(t, !tt.wantPurge, exists)
		})
	}
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

				exists, err := repo.db.NewSelect().Model((*types.JWKS)(nil)).Where("id = ?", key.ID).Exists(ctx)
				require.NoError(t, err)
				require.False(t, exists)
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
