package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestBunFlowStateRepository_ConsumeIsSingleUse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunFlowStateRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestFlowState("state-hash", now.Add(5*time.Minute)))
	require.NoError(t, err)

	found, err := repo.GetByStateHash(ctx, "state-hash")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "encrypted-verifier", found.CodeVerifierEncrypted)

	consumed, err := repo.Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeAuthorization, now)
	require.NoError(t, err)
	require.NotNil(t, consumed)
	require.NotNil(t, consumed.ConsumedAt)

	reused, err := repo.Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeAuthorization, now)
	require.NoError(t, err)
	assert.Nil(t, reused)
}

func TestBunFlowStateRepository_ConsumeRejectsWrongProviderPurposeOrExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunFlowStateRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestFlowState("state-hash", now.Add(5*time.Minute)))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestFlowState("expired", now.Add(-time.Minute)))
	require.NoError(t, err)

	wrongProvider, err := repo.Consume(ctx, "state-hash", "github", types.OAuthFlowPurposeAuthorization, now)
	require.NoError(t, err)
	assert.Nil(t, wrongProvider)

	wrongPurpose, err := repo.Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeCallback, now)
	require.NoError(t, err)
	assert.Nil(t, wrongPurpose)

	expired, err := repo.Consume(ctx, "expired", "google", types.OAuthFlowPurposeAuthorization, now)
	require.NoError(t, err)
	assert.Nil(t, expired)

	valid, err := repo.Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeAuthorization, now)
	require.NoError(t, err)
	require.NotNil(t, valid)
}

func TestBunFlowStateRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunFlowStateRepository(newTestOAuth2DB(t))
	now := testNow()

	_, err := repo.Create(ctx, createTestFlowState("expired", now.Add(-time.Minute)))
	require.NoError(t, err)
	_, err = repo.Create(ctx, createTestFlowState("valid", now.Add(time.Minute)))
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(ctx, now)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	expired, err := repo.GetByStateHash(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, expired)

	valid, err := repo.GetByStateHash(ctx, "valid")
	require.NoError(t, err)
	assert.NotNil(t, valid)
}
