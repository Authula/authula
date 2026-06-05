package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

func setupRefreshTokenRepo(t *testing.T) (*bun.DB, *refreshTokenRepositoryImpl) {
	t.Helper()
	db := internaltests.NewSQLiteIntegrationDB(t)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMP NOT NULL,
		is_revoked INTEGER DEFAULT 0,
		revoked_at TIMESTAMP NULL,
		last_reuse_attempt TIMESTAMP NULL DEFAULT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)
	return db, &refreshTokenRepositoryImpl{db: db}
}

func TestRefreshTokenRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("StoreRefreshToken", func(t *testing.T) {
		_, repo := setupRefreshTokenRepo(t)
		now := time.Now().Truncate(time.Millisecond)
		record := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}
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
	})

	t.Run("GetRefreshToken_not_found", func(t *testing.T) {
		_, repo := setupRefreshTokenRepo(t)
		got, err := repo.GetRefreshToken(ctx, "nonexistent-hash")
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("RevokeRefreshToken", func(t *testing.T) {
		_, repo := setupRefreshTokenRepo(t)
		now := time.Now().Truncate(time.Millisecond)
		record := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}
		err := repo.StoreRefreshToken(ctx, record)
		require.NoError(t, err)

		err = repo.RevokeRefreshToken(ctx, record.TokenHash)
		require.NoError(t, err)

		got, err := repo.GetRefreshToken(ctx, record.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.True(t, got.IsRevoked)
		require.NotNil(t, got.RevokedAt)
	})

	t.Run("RevokeAllSessionTokens", func(t *testing.T) {
		_, repo := setupRefreshTokenRepo(t)
		now := time.Now().Truncate(time.Millisecond)
		sessionID := uuid.New().String()
		otherSessionID := uuid.New().String()

		token1 := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}
		token2 := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}
		token3 := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: otherSessionID,
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}

		require.NoError(t, repo.StoreRefreshToken(ctx, token1))
		require.NoError(t, repo.StoreRefreshToken(ctx, token2))
		require.NoError(t, repo.StoreRefreshToken(ctx, token3))

		err := repo.RevokeAllSessionTokens(ctx, sessionID)
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

	t.Run("SetLastReuseAttempt", func(t *testing.T) {
		_, repo := setupRefreshTokenRepo(t)
		now := time.Now().Truncate(time.Millisecond)
		record := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}
		err := repo.StoreRefreshToken(ctx, record)
		require.NoError(t, err)

		err = repo.SetLastReuseAttempt(ctx, record.TokenHash)
		require.NoError(t, err)

		got, err := repo.GetRefreshToken(ctx, record.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotNil(t, got.LastReuseAttempt)
	})

	t.Run("CleanupExpiredTokens", func(t *testing.T) {
		_, repo := setupRefreshTokenRepo(t)
		now := time.Now().Truncate(time.Millisecond)

		expired := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(-time.Hour),
		}
		valid := &types.RefreshToken{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			TokenHash: uuid.New().String(),
			ExpiresAt: now.Add(time.Hour),
		}

		require.NoError(t, repo.StoreRefreshToken(ctx, expired))
		require.NoError(t, repo.StoreRefreshToken(ctx, valid))

		err := repo.CleanupExpiredTokens(ctx)
		require.NoError(t, err)

		gotExpired, err := repo.GetRefreshToken(ctx, expired.TokenHash)
		require.NoError(t, err)
		require.Nil(t, gotExpired)

		gotValid, err := repo.GetRefreshToken(ctx, valid.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, gotValid)
	})
}
