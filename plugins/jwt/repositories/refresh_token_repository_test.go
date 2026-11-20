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

func newRefreshTokenRecord() *types.RefreshToken {
	return &types.RefreshToken{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		TokenHash: uuid.New().String(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestNewRefreshTokenRepository(t *testing.T) {
	t.Parallel()

	db, _ := setupRefreshTokenRepo(t)
	repo := NewRefreshTokenRepository(db)
	require.NotNil(t, repo)

	ctx := context.Background()
	record := newRefreshTokenRecord()

	require.NoError(t, repo.StoreRefreshToken(ctx, record))

	got, err := repo.GetRefreshToken(ctx, record.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, record.ID, got.ID)
}

func TestRefreshTokenRepository_StoreDuplicate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(first, second *types.RefreshToken)
		wantMsg string
	}{
		{
			name: "rejects a duplicate token hash",
			mutate: func(first, second *types.RefreshToken) {
				second.TokenHash = first.TokenHash
			},
			wantMsg: "failed to store refresh token",
		},
		{
			name: "rejects a duplicate primary key",
			mutate: func(first, second *types.RefreshToken) {
				second.ID = first.ID
			},
			wantMsg: "failed to store refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			first := newRefreshTokenRecord()
			require.NoError(t, repo.StoreRefreshToken(ctx, first))

			second := newRefreshTokenRecord()
			tt.mutate(first, second)

			err := repo.StoreRefreshToken(ctx, second)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestRefreshTokenRepository_DatabaseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		call       func(ctx context.Context, repo *refreshTokenRepositoryImpl) error
		wantErrMsg string
	}{
		{
			name: "StoreRefreshToken wraps the insert failure",
			call: func(ctx context.Context, repo *refreshTokenRepositoryImpl) error {
				return repo.StoreRefreshToken(ctx, newRefreshTokenRecord())
			},
			wantErrMsg: "failed to store refresh token",
		},
		{
			name: "GetRefreshToken wraps the select failure",
			call: func(ctx context.Context, repo *refreshTokenRepositoryImpl) error {
				_, err := repo.GetRefreshToken(ctx, "any-hash")
				return err
			},
			wantErrMsg: "failed to get refresh token",
		},
		{
			name: "RevokeAllSessionTokens wraps the update failure",
			call: func(ctx context.Context, repo *refreshTokenRepositoryImpl) error {
				return repo.RevokeAllSessionTokens(ctx, "any-session")
			},
			wantErrMsg: "failed to revoke session tokens",
		},
		{
			name: "CleanupExpiredTokens wraps the delete failure",
			call: func(ctx context.Context, repo *refreshTokenRepositoryImpl) error {
				return repo.CleanupExpiredTokens(ctx)
			},
			wantErrMsg: "failed to cleanup expired tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, repo := setupRefreshTokenRepo(t)
			require.NoError(t, db.Close())

			err := tt.call(context.Background(), repo)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
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

func TestRefreshTokenRepository_RotateRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		revokeFirst       bool
		conflictingInsert bool
		wantErr           bool
		wantRotated       bool
		wantNewStored     bool
		wantOldRevoked    bool
	}{
		{
			name:           "rotates an active token",
			wantRotated:    true,
			wantNewStored:  true,
			wantOldRevoked: true,
		},
		{
			name:           "reports a conflict when already revoked",
			revokeFirst:    true,
			wantRotated:    false,
			wantNewStored:  false,
			wantOldRevoked: true,
		},
		{
			name:              "rolls back the revocation when the insert fails",
			conflictingInsert: true,
			wantErr:           true,
			wantRotated:       false,
			wantNewStored:     false,
			wantOldRevoked:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			oldToken := &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: uuid.New().String(),
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			}
			require.NoError(t, repo.StoreRefreshToken(ctx, oldToken))
			if tt.revokeFirst {
				_, err := repo.db.NewUpdate().
					Model(&types.RefreshToken{}).
					Where("token_hash = ?", oldToken.TokenHash).
					Set("is_revoked = ?, revoked_at = ?", true, time.Now()).
					Exec(ctx)
				require.NoError(t, err)
			}

			newToken := &types.RefreshToken{
				ID:        uuid.New().String(),
				SessionID: oldToken.SessionID,
				TokenHash: uuid.New().String(),
				ExpiresAt: time.Now().Add(time.Hour),
			}
			if tt.conflictingInsert {
				// Reusing the old primary key makes the insert fail inside the transaction.
				newToken.ID = oldToken.ID
			}

			rotated, err := repo.RotateRefreshToken(ctx, oldToken.TokenHash, newToken)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantRotated, rotated)

			gotOld, err := repo.GetRefreshToken(ctx, oldToken.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, gotOld)
			require.Equal(t, tt.wantOldRevoked, gotOld.IsRevoked)
			if tt.wantOldRevoked {
				require.NotNil(t, gotOld.RevokedAt)
			} else {
				require.Nil(t, gotOld.RevokedAt)
			}

			gotNew, err := repo.GetRefreshToken(ctx, newToken.TokenHash)
			require.NoError(t, err)
			if tt.wantNewStored {
				require.NotNil(t, gotNew)
				require.False(t, gotNew.IsRevoked)
			} else {
				require.Nil(t, gotNew)
			}
		})
	}
}

func TestRefreshTokenRepository_RotateRefreshTokenErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		replace     *types.RefreshToken
		blockUpdate bool
		wantErrMsg  string
	}{
		{
			name:       "rejects a nil replacement record",
			replace:    nil,
			wantErrMsg: "new refresh token record is required",
		},
		{
			name:        "rolls back when revoking the old token fails",
			replace:     newRefreshTokenRecord(),
			blockUpdate: true,
			wantErrMsg:  "failed to revoke old refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, repo := setupRefreshTokenRepo(t)
			ctx := context.Background()

			oldToken := newRefreshTokenRecord()
			require.NoError(t, repo.StoreRefreshToken(ctx, oldToken))

			if tt.blockUpdate {
				_, err := db.NewRaw(
					`CREATE TRIGGER block_refresh_token_update BEFORE UPDATE ON refresh_tokens
					BEGIN SELECT RAISE(ABORT, 'update blocked'); END;`,
				).Exec(ctx)
				require.NoError(t, err)
			}

			rotated, err := repo.RotateRefreshToken(ctx, oldToken.TokenHash, tt.replace)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrMsg)
			require.False(t, rotated)

			gotOld, err := repo.GetRefreshToken(ctx, oldToken.TokenHash)
			require.NoError(t, err)
			require.NotNil(t, gotOld)
			require.False(t, gotOld.IsRevoked)
			require.Nil(t, gotOld.RevokedAt)
		})
	}
}

func TestRefreshTokenRepository_RotateRefreshTokenOnlyOnce(t *testing.T) {
	t.Parallel()

	_, repo := setupRefreshTokenRepo(t)
	ctx := context.Background()

	oldToken := newRefreshTokenRecord()
	require.NoError(t, repo.StoreRefreshToken(ctx, oldToken))

	first := newRefreshTokenRecord()
	rotated, err := repo.RotateRefreshToken(ctx, oldToken.TokenHash, first)
	require.NoError(t, err)
	require.True(t, rotated)

	second := newRefreshTokenRecord()
	rotated, err = repo.RotateRefreshToken(ctx, oldToken.TokenHash, second)
	require.NoError(t, err)
	require.False(t, rotated)

	notStored, err := repo.GetRefreshToken(ctx, second.TokenHash)
	require.NoError(t, err)
	require.Nil(t, notStored)
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
