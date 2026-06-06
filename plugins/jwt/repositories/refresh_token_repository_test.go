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

func setupRefreshTokenRepo(t *testing.T) (*bun.DB, *refreshTokenRepositoryImpl) {
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
	return db, &refreshTokenRepositoryImpl{db: db}
}

func TestRefreshTokenRepository_Store(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record func() *types.RefreshToken
	}{
		{
			name: "stores and retrieves refresh token",
			record: func() *types.RefreshToken {
				return &types.RefreshToken{
					ID:        uuid.New().String(),
					SessionID: uuid.New().String(),
					TokenHash: uuid.New().String(),
					ExpiresAt: time.Now().Add(time.Hour),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			record := tt.record()
			now := time.Now().Truncate(time.Millisecond)

			err := repo.StoreRefreshToken(ctx, record)
			require.NoError(t, err)

			got, err := repo.GetRefreshToken(ctx, record.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, record.ID, got.ID)
			require.Equal(t, record.SessionID, got.SessionID)
			require.Equal(t, record.TokenHash, got.TokenHash)
			require.WithinDuration(t, record.ExpiresAt, got.ExpiresAt, time.Second)
			require.False(t, got.IsRevoked)
			require.Nil(t, got.RevokedAt)
			require.Nil(t, got.LastReuseAttempt)
			require.WithinDuration(t, now, got.CreatedAt, time.Second)
		})
	}
}

func TestRefreshTokenRepository_GetRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tokenHash string
		wantNil   bool
	}{
		{
			name:      "returns token when found",
			tokenHash: uuid.New().String(),
			wantNil:   false,
		},
		{
			name:      "returns nil when not found",
			tokenHash: "nonexistent-hash",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			if !tt.wantNil {
				err := repo.StoreRefreshToken(ctx, &types.RefreshToken{
					ID:        uuid.New().String(),
					SessionID: uuid.New().String(),
					TokenHash: tt.tokenHash,
					ExpiresAt: time.Now().Add(time.Hour),
				})
				require.NoError(t, err)
			}

			got, err := repo.GetRefreshToken(ctx, tt.tokenHash)

			require.NoError(t, err)
			if tt.wantNil {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, tt.tokenHash, got.TokenHash)
			}
		})
	}
}

func TestRefreshTokenRepository_RevokeRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record *types.RefreshToken
	}{
		{
			name: "revokes token and sets revoked_at",
			record: &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: uuid.New().String(),
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			err := repo.StoreRefreshToken(ctx, tt.record)
			require.NoError(t, err)

			err = repo.RevokeRefreshToken(ctx, tt.record.TokenHash)
			require.NoError(t, err)

			got, err := repo.GetRefreshToken(ctx, tt.record.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.True(t, got.IsRevoked)
			require.NotNil(t, got.RevokedAt)
		})
	}
}

func TestRefreshTokenRepository_RevokeAllSessionTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		targetSession string
		otherSession  string
	}{
		{
			name:          "revokes only tokens for the target session",
			targetSession: uuid.New().String(),
			otherSession:  uuid.New().String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			token1 := &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: tt.targetSession,
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			}
			token2 := &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: tt.targetSession,
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			}
			token3 := &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: tt.otherSession,
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			}

			require.NoError(t, repo.StoreRefreshToken(ctx, token1))
			require.NoError(t, repo.StoreRefreshToken(ctx, token2))
			require.NoError(t, repo.StoreRefreshToken(ctx, token3))

			err := repo.RevokeAllSessionTokens(ctx, tt.targetSession)
			require.NoError(t, err)

			got1, err := repo.GetRefreshToken(ctx, token1.TokenHash)
			require.NoError(t, err)
			require.True(t, got1.IsRevoked)
			require.NotNil(t, got1.RevokedAt)

			got2, err := repo.GetRefreshToken(ctx, token2.TokenHash)
			require.NoError(t, err)
			require.True(t, got2.IsRevoked)
			require.NotNil(t, got2.RevokedAt)

			got3, err := repo.GetRefreshToken(ctx, token3.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, got3)
			require.False(t, got3.IsRevoked)
			require.Nil(t, got3.RevokedAt)
		})
	}
}

func TestRefreshTokenRepository_SetLastReuseAttempt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record *types.RefreshToken
	}{
		{
			name: "sets last_reuse_attempt on token",
			record: &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: uuid.New().String(),
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			err := repo.StoreRefreshToken(ctx, tt.record)
			require.NoError(t, err)

			err = repo.SetLastReuseAttempt(ctx, tt.record.TokenHash)
			require.NoError(t, err)

			got, err := repo.GetRefreshToken(ctx, tt.record.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.LastReuseAttempt)
		})
	}
}

func TestRefreshTokenRepository_CleanupExpiredTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		expiredRecord *types.RefreshToken
		validRecord   *types.RefreshToken
	}{
		{
			name: "removes expired tokens and keeps valid ones",
			expiredRecord: &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: uuid.New().String(),
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			validRecord: &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: uuid.New().String(),
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			require.NoError(t, repo.StoreRefreshToken(ctx, tt.expiredRecord))
			require.NoError(t, repo.StoreRefreshToken(ctx, tt.validRecord))

			err := repo.CleanupExpiredTokens(ctx)
			require.NoError(t, err)

			gotExpired, err := repo.GetRefreshToken(ctx, tt.expiredRecord.TokenHash)
			require.NoError(t, err)
			require.Nil(t, gotExpired)

			gotValid, err := repo.GetRefreshToken(ctx, tt.validRecord.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, gotValid)
		})
	}
}
