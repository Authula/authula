package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBunAuthorizationCodeRepository_ConsumeIsSingleUse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestAuthorizationCode("hash-1", "client-1", now.Add(time.Minute)))
	require.NoError(t, err)

	found, err := repo.GetByCodeHash(ctx, "hash-1")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Nil(t, found.ConsumedAt)

	consumed, err := repo.Consume(ctx, "hash-1", "client-1", now)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	require.NotNil(t, consumed.ConsumedAt)

	reused, err := repo.Consume(ctx, "hash-1", "client-1", now)
	require.NoError(t, err)
	assert.Nil(t, reused)
}

func TestBunAuthorizationCodeRepository_ConsumeRejectsWrongClientAndExpiredCode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestAuthorizationCode("hash-1", "client-1", now.Add(time.Minute)))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestAuthorizationCode("hash-2", "client-1", now.Add(-time.Minute)))
	require.NoError(t, err)

	wrongClient, err := repo.Consume(ctx, "hash-1", "client-2", now)
	require.NoError(t, err)
	assert.Nil(t, wrongClient)

	expired, err := repo.Consume(ctx, "hash-2", "client-1", now)
	require.NoError(t, err)
	assert.Nil(t, expired)

	stillValid, err := repo.Consume(ctx, "hash-1", "client-1", now)
	require.NoError(t, err)
	require.NotNil(t, stillValid)
}

func TestBunAuthorizationCodeRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestAuthorizationCode("expired", "client-1", now.Add(-time.Minute)))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestAuthorizationCode("valid", "client-1", now.Add(time.Minute)))
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(ctx, now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	expired, err := repo.GetByCodeHash(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, expired)

	valid, err := repo.GetByCodeHash(ctx, "valid")
	require.NoError(t, err)
	assert.NotNil(t, valid)
}
