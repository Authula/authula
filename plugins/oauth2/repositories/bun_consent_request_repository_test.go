package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBunConsentRequestRepository_ConsumeIsSingleUse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestConsentRequest("request-hash", "user-1", "client-1", now.Add(10*time.Minute)))
	require.NoError(t, err)

	found, err := repo.GetByRequestHash(ctx, "request-hash")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Nil(t, found.ConsumedAt)

	consumed, err := repo.Consume(ctx, "request-hash", now)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	require.NotNil(t, consumed.ConsumedAt)

	reused, err := repo.Consume(ctx, "request-hash", now)
	require.NoError(t, err)
	assert.Nil(t, reused)
}

func TestBunConsentRequestRepository_ConsumeRejectsExpiredRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestConsentRequest("expired", "user-1", "client-1", now.Add(-time.Minute)))
	require.NoError(t, err)

	consumed, err := repo.Consume(ctx, "expired", now)
	require.NoError(t, err)
	assert.Nil(t, consumed)
}

func TestBunConsentRequestRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestConsentRequest("expired", "user-1", "client-1", now.Add(-time.Minute)))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestConsentRequest("valid", "user-1", "client-1", now.Add(time.Minute)))
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(ctx, now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	expired, err := repo.GetByRequestHash(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, expired)

	valid, err := repo.GetByRequestHash(ctx, "valid")
	require.NoError(t, err)
	assert.NotNil(t, valid)
}
