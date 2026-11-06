package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestBunConsentRepository_ActiveConsentLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid profile", now))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestConsent("consent-2", "user-1", "client-2", "openid email", now))
	require.NoError(t, err)

	active, err := repo.GetActiveByUserAndClient(ctx, "user-1", "client-1")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, "consent-1", active.ID)

	consents, err := repo.ListActiveByUser(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, consents, 2)

	revoked, err := repo.Revoke(ctx, "consent-1", now)
	require.NoError(t, err)
	assert.True(t, revoked)

	active, err = repo.GetActiveByUserAndClient(ctx, "user-1", "client-1")
	require.NoError(t, err)
	assert.Nil(t, active)

	consents, err = repo.ListActiveByUser(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, consents, 1)

	revoked, err = repo.Revoke(ctx, "consent-1", now)
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestBunConsentRepository_ScopeChangeRevokesPreviousConsent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", now))
	require.NoError(t, err)

	revoked, err := repo.RevokeByUserAndClient(ctx, "user-1", "client-1", now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, revoked)

	_, err = repo.Create(ctx, createTestConsent("consent-2", "user-1", "client-1", "openid profile", now))
	require.NoError(t, err)

	active, err := repo.GetActiveByUserAndClient(ctx, "user-1", "client-1")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, "consent-2", active.ID)
	assert.Equal(t, "openid profile", active.Scope)
}

func TestBunConsentRepository_RevokeByClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", now))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestConsent("consent-2", "user-2", "client-1", "openid", now))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestConsent("consent-3", "user-3", "client-2", "openid", now))
	require.NoError(t, err)

	revoked, err := repo.RevokeByClient(ctx, "client-1", now)
	require.NoError(t, err)
	assert.EqualValues(t, 2, revoked)

	consent, err := repo.GetByID(ctx, "consent-1")
	require.NoError(t, err)
	require.NotNil(t, consent)
	assert.NotNil(t, consent.RevokedAt)
	assert.Equal(t, types.OAuthConsentSourceExplicit, consent.Source)
}
