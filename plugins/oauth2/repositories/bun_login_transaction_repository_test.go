package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBunLoginTransactionRepository_CompleteAndConsume(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestLoginTransaction("transaction-hash", now.Add(10*time.Minute)))
	require.NoError(t, err)

	found, err := repo.GetByTransactionHash(ctx, "transaction-hash")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Nil(t, found.CompletedAt)

	completed, err := repo.MarkCompleted(ctx, "transaction-hash", now)
	require.NoError(t, err)
	assert.True(t, completed)

	completed, err = repo.MarkCompleted(ctx, "transaction-hash", now)
	require.NoError(t, err)
	assert.False(t, completed)

	consumed, err := repo.Consume(ctx, "transaction-hash", now)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	require.NotNil(t, consumed.ConsumedAt)

	reused, err := repo.Consume(ctx, "transaction-hash", now)
	require.NoError(t, err)
	assert.Nil(t, reused)
}

func TestBunLoginTransactionRepository_ConsumeRejectsExpiredTransaction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestLoginTransaction("expired", now.Add(-time.Minute)))
	require.NoError(t, err)

	consumed, err := repo.Consume(ctx, "expired", now)
	require.NoError(t, err)
	assert.Nil(t, consumed)
}

func TestBunLoginTransactionRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestLoginTransaction("expired", now.Add(-time.Minute)))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestLoginTransaction("valid", now.Add(time.Minute)))
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(ctx, now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	expired, err := repo.GetByTransactionHash(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, expired)

	valid, err := repo.GetByTransactionHash(ctx, "valid")
	require.NoError(t, err)
	assert.NotNil(t, valid)
}
