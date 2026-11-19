package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestBunAccessGrantRepository_CRUDAndRevoke(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAccessGrantRepository(newTestOAuth2DB(t))
	grant := createTestAccessGrant("jti-1", "family-1", testNow().Add(time.Hour))

	created, err := repo.Create(ctx, grant)
	require.NoError(t, err)
	assert.Equal(t, grant.ID, created.ID)

	byJTI, err := repo.GetByJTI(ctx, "jti-1")
	require.NoError(t, err)
	require.NotNil(t, byJTI)
	assert.Equal(t, grant.ID, byJTI.ID)

	byID, err := repo.GetByID(ctx, grant.ID)
	require.NoError(t, err)
	require.NotNil(t, byID)

	revoked, err := repo.Revoke(ctx, "jti-1", testNow())
	require.NoError(t, err)
	assert.True(t, revoked)

	revoked, err = repo.Revoke(ctx, "jti-1", testNow())
	require.NoError(t, err)
	assert.False(t, revoked)

	stored, err := repo.GetByJTI(ctx, "jti-1")
	require.NoError(t, err)
	require.NotNil(t, stored.RevokedAt)
}

func TestBunAccessGrantRepository_RevokeFamilyAndDeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAccessGrantRepository(newTestOAuth2DB(t))
	for _, grant := range []*types.OAuthAccessGrant{
		createTestAccessGrant("jti-1", "family-1", testNow().Add(time.Hour)),
		createTestAccessGrant("jti-2", "family-1", testNow().Add(time.Hour)),
		createTestAccessGrant("jti-3", "family-2", testNow().Add(time.Hour)),
		createTestAccessGrant("jti-4", "family-2", testNow().Add(-time.Minute)),
	} {
		_, err := repo.Create(ctx, grant)
		require.NoError(t, err)
	}

	revoked, err := repo.RevokeFamily(ctx, "family-1", testNow())
	require.NoError(t, err)
	assert.Equal(t, int64(2), revoked)

	deleted, err := repo.DeleteExpired(ctx, testNow())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}
