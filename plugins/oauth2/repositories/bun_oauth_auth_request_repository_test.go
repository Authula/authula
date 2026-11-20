package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestBunAuthRequestRepository_CRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAuthRequestRepository(newTestOAuth2DB(t))
	request := createTestAuthRequest("binding-1", "client-1", testNow().Add(10*time.Minute))

	created, err := repo.Create(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, request.ID, created.ID)

	byID, err := repo.GetByID(ctx, request.ID)
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, request.BindingHash, byID.BindingHash)

	byBinding, err := repo.GetByBindingHash(ctx, "binding-1")
	require.NoError(t, err)
	require.NotNil(t, byBinding)
	assert.Equal(t, request.ID, byBinding.ID)

	missing, err := repo.GetByID(ctx, "missing")
	require.ErrorIs(t, err, ErrAuthRequestNotFound)
	assert.Nil(t, missing)
}

func TestBunAuthRequestRepository_ConsumeCode(t *testing.T) {
	t.Parallel()

	now := testNow()

	tests := []struct {
		name     string
		request  *types.OAuthAuthRequest
		clientID string
		wantNil  bool
	}{
		{
			name: "consumes an issued code once",
			request: func() *types.OAuthAuthRequest {
				r := createTestAuthRequest("binding-1", "client-1", now.Add(10*time.Minute))
				r.CodeHash = new("code-hash")
				r.CodeIssuedAt = new(now)
				r.CompletedAt = new(now)
				return r
			}(),
			clientID: "client-1",
		},
		{
			name: "rejects an expired code",
			request: func() *types.OAuthAuthRequest {
				r := createTestAuthRequest("binding-2", "client-1", now.Add(-time.Minute))
				r.CodeHash = new("expired-hash")
				r.CodeIssuedAt = new(now)
				r.CompletedAt = new(now)
				return r
			}(),
			clientID: "client-1",
			wantNil:  true,
		},
		{
			name: "rejects a code that was never issued",
			request: func() *types.OAuthAuthRequest {
				r := createTestAuthRequest("binding-3", "client-1", now.Add(10*time.Minute))
				r.CodeHash = new("unissued-hash")
				return r
			}(),
			clientID: "client-1",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunAuthRequestRepository(newTestOAuth2DB(t))
			_, err := repo.Create(ctx, tt.request)
			require.NoError(t, err)

			consumed, err := repo.ConsumeCode(ctx, *tt.request.CodeHash, tt.clientID, now)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, consumed)
				return
			}
			require.NotNil(t, consumed)
			require.NotNil(t, consumed.CodeConsumedAt)

			replay, err := repo.ConsumeCode(ctx, *tt.request.CodeHash, tt.clientID, now)
			require.NoError(t, err)
			assert.Nil(t, replay, "a code must only be consumed once")
		})
	}
}

func TestBunAuthRequestRepository_ConsumeCodeWrongClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := testNow()
	repo := NewBunAuthRequestRepository(newTestOAuth2DB(t))
	request := createTestAuthRequest("binding-1", "client-1", now.Add(10*time.Minute))
	request.CodeHash = new("code-hash")
	request.CodeIssuedAt = &now
	_, err := repo.Create(ctx, request)
	require.NoError(t, err)

	consumed, err := repo.ConsumeCode(ctx, "code-hash", "client-2", now)
	require.NoError(t, err)
	assert.Nil(t, consumed)

	consumed, err = repo.ConsumeCode(ctx, "code-hash", "client-1", now)
	require.NoError(t, err)
	assert.NotNil(t, consumed)
}

func TestBunAuthRequestRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAuthRequestRepository(newTestOAuth2DB(t))
	expired := createTestAuthRequest("expired", "client-1", testNow().Add(-time.Minute))
	valid := createTestAuthRequest("valid", "client-1", testNow().Add(time.Minute))
	_, err := repo.Create(ctx, expired)
	require.NoError(t, err)
	_, err = repo.Create(ctx, valid)
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(ctx, testNow())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	stored, err := repo.GetByID(ctx, valid.ID)
	require.NoError(t, err)
	assert.NotNil(t, stored)
}
