package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestBunRefreshTokenRepository_CRUDAndRotate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunRefreshTokenRepository(newTestOAuth2DB(t))
	now := testNow()
	token := createTestRefreshToken("hash-1", "family-1", now.Add(time.Hour))
	_, err := repo.Create(ctx, token)
	require.NoError(t, err)

	stored, err := repo.GetByTokenHash(ctx, "hash-1")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, token.ID, stored.ID)

	replacement := createTestRefreshToken("hash-2", "family-1", now.Add(2*time.Hour))
	replacement.RotatedFrom = new("hash-1")
	consumed, err := repo.Rotate(ctx, "hash-1", replacement, now)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	require.NotNil(t, consumed.ConsumedAt)

	// Replaying the old token must not rotate again.
	replay := createTestRefreshToken("hash-3", "family-1", now.Add(3*time.Hour))
	consumed, err = repo.Rotate(ctx, "hash-1", replay, now)
	require.NoError(t, err)
	assert.Nil(t, consumed)

	newToken, err := repo.GetByTokenHash(ctx, "hash-2")
	require.NoError(t, err)
	require.NotNil(t, newToken)
	assert.Equal(t, "hash-1", *newToken.RotatedFrom)
}

func TestBunRefreshTokenRepository_RotateRejectsExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunRefreshTokenRepository(newTestOAuth2DB(t))
	now := testNow()
	token := createTestRefreshToken("expired", "family-1", now.Add(-time.Minute))
	_, err := repo.Create(ctx, token)
	require.NoError(t, err)

	replacement := createTestRefreshToken("new", "family-1", now.Add(time.Hour))
	consumed, err := repo.Rotate(ctx, "expired", replacement, now)
	require.NoError(t, err)
	assert.Nil(t, consumed)
}

func TestBunRefreshTokenRepository_RevokeFamilyAndDeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunRefreshTokenRepository(newTestOAuth2DB(t))
	now := testNow()
	for _, token := range []*types.OAuthRefreshToken{
		createTestRefreshToken("hash-1", "family-1", now.Add(time.Hour)),
		createTestRefreshToken("hash-2", "family-1", now.Add(time.Hour)),
		createTestRefreshToken("hash-3", "family-2", now.Add(time.Hour)),
		createTestRefreshToken("hash-4", "family-2", now.Add(-time.Minute)),
	} {
		_, err := repo.Create(ctx, token)
		require.NoError(t, err)
	}

	revoked, err := repo.RevokeFamily(ctx, "family-1", now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), revoked)

	// Revoking again does not double count already-revoked rows.
	revoked, err = repo.RevokeFamily(ctx, "family-1", now)
	require.NoError(t, err)
	assert.Zero(t, revoked)

	deleted, err := repo.DeleteExpired(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}
