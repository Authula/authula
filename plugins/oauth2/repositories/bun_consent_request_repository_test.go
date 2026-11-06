package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func assertConsentRequestEqual(t *testing.T, want *types.OAuthConsentRequest, got *types.OAuthConsentRequest) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.RequestHash, got.RequestHash)
	assert.Equal(t, want.ClientID, got.ClientID)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.SessionID, got.SessionID)
	assert.Equal(t, want.RedirectURI, got.RedirectURI)
	assert.Equal(t, want.Scope, got.Scope)
	assert.Equal(t, want.State, got.State)
	assert.Equal(t, want.Nonce, got.Nonce)
	assert.Equal(t, want.CodeChallenge, got.CodeChallenge)
	assert.Equal(t, want.CodeChallengeMethod, got.CodeChallengeMethod)
	assert.Equal(t, want.Prompt, got.Prompt)
	assert.Equal(t, want.MaxAge, got.MaxAge)
	assert.True(t, want.AuthTime.Equal(got.AuthTime))
	assert.True(t, want.ExpiresAt.Equal(got.ExpiresAt))
	assert.Equal(t, want.ConsumedAt, got.ConsumedAt)
}

func TestBunConsentRequestRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []*types.OAuthConsentRequest
		request  *types.OAuthConsentRequest
		wantErr  bool
	}{
		{
			name:    "creates and returns the request",
			request: createTestConsentRequest("request-hash", "user-1", "client-1", testNow().Add(10*time.Minute)),
		},
		{
			name: "rejects a duplicate request hash",
			existing: []*types.OAuthConsentRequest{
				createTestConsentRequest("request-hash", "user-1", "client-1", testNow().Add(10*time.Minute)),
			},
			request: createTestConsentRequest("request-hash", "user-2", "client-2", testNow().Add(10*time.Minute)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))

			for _, request := range tt.existing {
				_, err := repo.Create(ctx, request)
				require.NoError(t, err)
			}

			created, err := repo.Create(ctx, tt.request)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertConsentRequestEqual(t, tt.request, created)

			stored, err := repo.GetByRequestHash(ctx, tt.request.RequestHash)
			require.NoError(t, err)
			assertConsentRequestEqual(t, tt.request, stored)
		})
	}
}

func TestBunConsentRequestRepository_GetByRequestHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requests    []*types.OAuthConsentRequest
		requestHash string
		wantID      string
	}{
		{
			name: "returns the matching request",
			requests: []*types.OAuthConsentRequest{
				createTestConsentRequest("request-hash-1", "user-1", "client-1", testNow().Add(10*time.Minute)),
				createTestConsentRequest("request-hash-2", "user-2", "client-2", testNow().Add(10*time.Minute)),
			},
			requestHash: "request-hash-2",
			wantID:      "request-request-hash-2",
		},
		{
			name:        "returns nil for a missing request",
			requestHash: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))

			for _, request := range tt.requests {
				_, err := repo.Create(ctx, request)
				require.NoError(t, err)
			}

			found, err := repo.GetByRequestHash(ctx, tt.requestHash)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, found)
				return
			}

			require.NotNil(t, found)
			assert.Equal(t, tt.wantID, found.ID)
			assert.Equal(t, tt.requestHash, found.RequestHash)
		})
	}
}

func TestBunConsentRequestRepository_Consume(t *testing.T) {
	t.Parallel()

	type consumeAttempt struct {
		requestHash string
		wantNil     bool
	}

	tests := []struct {
		name     string
		requests []*types.OAuthConsentRequest
		attempts []consumeAttempt
	}{
		{
			name: "valid request can only be consumed once",
			requests: []*types.OAuthConsentRequest{
				createTestConsentRequest("request-hash", "user-1", "client-1", testNow().Add(10*time.Minute)),
			},
			attempts: []consumeAttempt{
				{requestHash: "request-hash"},
				{requestHash: "request-hash", wantNil: true},
			},
		},
		{
			name: "expired request is rejected",
			requests: []*types.OAuthConsentRequest{
				createTestConsentRequest("expired", "user-1", "client-1", testNow().Add(-time.Minute)),
			},
			attempts: []consumeAttempt{
				{requestHash: "expired", wantNil: true},
			},
		},
		{
			name: "missing request is rejected",
			attempts: []consumeAttempt{
				{requestHash: "missing", wantNil: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, request := range tt.requests {
				_, err := repo.Create(ctx, request)
				require.NoError(t, err)
			}

			for _, attempt := range tt.attempts {
				consumed, err := repo.Consume(ctx, attempt.requestHash, now)
				require.NoError(t, err)
				if attempt.wantNil {
					assert.Nil(t, consumed)
					continue
				}

				require.NotNil(t, consumed)
				require.NotNil(t, consumed.ConsumedAt)
				assert.Equal(t, attempt.requestHash, consumed.RequestHash)
			}
		})
	}
}

func TestBunConsentRequestRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		requests     []*types.OAuthConsentRequest
		wantDeleted  int64
		wantRetained []string
		wantRemoved  []string
	}{
		{
			name: "deletes only expired requests",
			requests: []*types.OAuthConsentRequest{
				createTestConsentRequest("expired", "user-1", "client-1", testNow().Add(-time.Minute)),
				createTestConsentRequest("valid", "user-1", "client-1", testNow().Add(time.Minute)),
			},
			wantDeleted:  1,
			wantRetained: []string{"valid"},
			wantRemoved:  []string{"expired"},
		},
		{
			name: "keeps all requests when none are expired",
			requests: []*types.OAuthConsentRequest{
				createTestConsentRequest("valid-1", "user-1", "client-1", testNow().Add(time.Minute)),
				createTestConsentRequest("valid-2", "user-1", "client-1", testNow().Add(time.Hour)),
			},
			wantDeleted:  0,
			wantRetained: []string{"valid-1", "valid-2"},
		},
		{
			name: "deletes all expired requests",
			requests: []*types.OAuthConsentRequest{
				createTestConsentRequest("expired-1", "user-1", "client-1", testNow().Add(-time.Hour)),
				createTestConsentRequest("expired-2", "user-1", "client-1", testNow().Add(-time.Minute)),
			},
			wantDeleted: 2,
			wantRemoved: []string{"expired-1", "expired-2"},
		},
		{
			name:        "handles an empty table",
			wantDeleted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRequestRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, request := range tt.requests {
				_, err := repo.Create(ctx, request)
				require.NoError(t, err)
			}

			deleted, err := repo.DeleteExpired(ctx, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDeleted, deleted)

			for _, requestHash := range tt.wantRetained {
				request, err := repo.GetByRequestHash(ctx, requestHash)
				require.NoError(t, err)
				assert.NotNil(t, request, "request %s should be retained", requestHash)
			}
			for _, requestHash := range tt.wantRemoved {
				request, err := repo.GetByRequestHash(ctx, requestHash)
				require.NoError(t, err)
				assert.Nil(t, request, "request %s should be deleted", requestHash)
			}
		})
	}
}

func TestBunConsentRequestRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rollback  bool
		wantStore bool
	}{
		{name: "commit persists the request", wantStore: true},
		{name: "rollback discards the request", rollback: true, wantStore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			repo := NewBunConsentRequestRepository(db)
			request := createTestConsentRequest("request-hash", "user-1", "client-1", testNow().Add(10*time.Minute))
			rollback := errors.New("rollback")

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := repo.WithTx(tx).Create(ctx, request); err != nil {
					return err
				}
				if tt.rollback {
					return rollback
				}
				return nil
			})
			if tt.rollback {
				require.ErrorIs(t, err, rollback)
			} else {
				require.NoError(t, err)
			}

			stored, err := repo.GetByRequestHash(ctx, request.RequestHash)
			require.NoError(t, err)
			if tt.wantStore {
				require.NotNil(t, stored)
				assert.Equal(t, request.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}
