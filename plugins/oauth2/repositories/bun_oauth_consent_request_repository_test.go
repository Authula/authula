package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestBunConsentRequestRepository_CRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
	request := createTestConsentRequest("token-hash", "user-1", "client-1", testNow().Add(10*time.Minute))
	request.CompletionCodeHash = new("completion-hash")

	_, err := repo.Create(ctx, request)
	require.NoError(t, err)

	byToken, err := repo.GetByTokenHash(ctx, "token-hash")
	require.NoError(t, err)
	require.NotNil(t, byToken)
	assert.Equal(t, request.ID, byToken.ID)

	byCode, err := repo.GetByCompletionCodeHash(ctx, "completion-hash")
	require.NoError(t, err)
	require.NotNil(t, byCode)
	assert.Equal(t, request.ID, byCode.ID)

	byID, err := repo.GetByID(ctx, request.ID)
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, types.OAuthConsentDecisionAllow, *byID.Decision)
}

func TestBunConsentRequestRepository_Consume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt time.Time
		wantNil   bool
	}{
		{name: "consumes a valid challenge", expiresAt: testNow().Add(10 * time.Minute)},
		{name: "rejects an expired challenge", expiresAt: testNow().Add(-time.Minute), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			now := testNow()
			repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
			request := createTestConsentRequest("token-hash", "user-1", "client-1", tt.expiresAt)
			_, err := repo.Create(ctx, request)
			require.NoError(t, err)

			consumed, err := repo.Consume(ctx, "token-hash", now)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, consumed)
				return
			}
			require.NotNil(t, consumed)
			require.NotNil(t, consumed.ConsumedAt)

			replay, err := repo.Consume(ctx, "token-hash", now)
			require.NoError(t, err)
			assert.Nil(t, replay, "a consent challenge must only be consumed once")
		})
	}
}

func TestBunConsentRequestRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
	expired := createTestConsentRequest("expired", "user-1", "client-1", testNow().Add(-time.Minute))
	valid := createTestConsentRequest("valid", "user-1", "client-1", testNow().Add(time.Minute))
	_, err := repo.Create(ctx, expired)
	require.NoError(t, err)
	_, err = repo.Create(ctx, valid)
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(ctx, testNow())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}
