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

func assertFlowStateEqual(t *testing.T, want *types.OAuthFlowState, got *types.OAuthFlowState) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.StateHash, got.StateHash)
	assert.Equal(t, want.Provider, got.Provider)
	assert.Equal(t, want.Purpose, got.Purpose)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.RedirectTo, got.RedirectTo)
	assert.Equal(t, want.CodeVerifierEncrypted, got.CodeVerifierEncrypted)
	assert.True(t, want.ExpiresAt.Equal(got.ExpiresAt))
	assert.Equal(t, want.ConsumedAt, got.ConsumedAt)
}

func TestBunFlowStateRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []*types.OAuthFlowState
		state    *types.OAuthFlowState
		wantErr  bool
	}{
		{
			name:  "creates and returns the state",
			state: createTestFlowState("state-hash", testNow().Add(5*time.Minute)),
		},
		{
			name: "rejects a duplicate state hash",
			existing: []*types.OAuthFlowState{
				createTestFlowState("state-hash", testNow().Add(5*time.Minute)),
			},
			state:   createTestFlowState("state-hash", testNow().Add(5*time.Minute)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunFlowStateRepository(newTestOAuth2DB(t))

			for _, state := range tt.existing {
				_, err := repo.Create(ctx, state)
				require.NoError(t, err)
			}

			created, err := repo.Create(ctx, tt.state)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertFlowStateEqual(t, tt.state, created)

			stored, err := repo.GetByStateHash(ctx, tt.state.StateHash)
			require.NoError(t, err)
			assertFlowStateEqual(t, tt.state, stored)
		})
	}
}

func TestBunFlowStateRepository_GetByStateHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		states    []*types.OAuthFlowState
		stateHash string
		wantID    string
	}{
		{
			name: "returns the matching state",
			states: []*types.OAuthFlowState{
				createTestFlowState("state-hash-1", testNow().Add(5*time.Minute)),
				createTestFlowState("state-hash-2", testNow().Add(5*time.Minute)),
			},
			stateHash: "state-hash-2",
			wantID:    "state-state-hash-2",
		},
		{
			name:      "returns nil for a missing state",
			stateHash: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunFlowStateRepository(newTestOAuth2DB(t))

			for _, state := range tt.states {
				_, err := repo.Create(ctx, state)
				require.NoError(t, err)
			}

			found, err := repo.GetByStateHash(ctx, tt.stateHash)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, found)
				return
			}

			require.NotNil(t, found)
			assert.Equal(t, tt.wantID, found.ID)
			assert.Equal(t, tt.stateHash, found.StateHash)
		})
	}
}

func TestBunFlowStateRepository_Consume(t *testing.T) {
	t.Parallel()

	type consumeAttempt struct {
		stateHash string
		provider  string
		purpose   string
		wantNil   bool
	}

	tests := []struct {
		name     string
		states   []*types.OAuthFlowState
		attempts []consumeAttempt
	}{
		{
			name: "valid state can only be consumed once",
			states: []*types.OAuthFlowState{
				createTestFlowState("state-hash", testNow().Add(5*time.Minute)),
			},
			attempts: []consumeAttempt{
				{stateHash: "state-hash", provider: "google", purpose: types.OAuthFlowPurposeAuthorization},
				{stateHash: "state-hash", provider: "google", purpose: types.OAuthFlowPurposeAuthorization, wantNil: true},
			},
		},
		{
			name: "wrong provider is rejected without consuming",
			states: []*types.OAuthFlowState{
				createTestFlowState("state-hash", testNow().Add(5*time.Minute)),
			},
			attempts: []consumeAttempt{
				{stateHash: "state-hash", provider: "github", purpose: types.OAuthFlowPurposeAuthorization, wantNil: true},
				{stateHash: "state-hash", provider: "google", purpose: types.OAuthFlowPurposeAuthorization},
			},
		},
		{
			name: "wrong purpose is rejected without consuming",
			states: []*types.OAuthFlowState{
				createTestFlowState("state-hash", testNow().Add(5*time.Minute)),
			},
			attempts: []consumeAttempt{
				{stateHash: "state-hash", provider: "google", purpose: types.OAuthFlowPurposeCallback, wantNil: true},
				{stateHash: "state-hash", provider: "google", purpose: types.OAuthFlowPurposeAuthorization},
			},
		},
		{
			name: "expired state is rejected",
			states: []*types.OAuthFlowState{
				createTestFlowState("expired", testNow().Add(-time.Minute)),
			},
			attempts: []consumeAttempt{
				{stateHash: "expired", provider: "google", purpose: types.OAuthFlowPurposeAuthorization, wantNil: true},
			},
		},
		{
			name: "missing state is rejected",
			attempts: []consumeAttempt{
				{stateHash: "missing", provider: "google", purpose: types.OAuthFlowPurposeAuthorization, wantNil: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunFlowStateRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, state := range tt.states {
				_, err := repo.Create(ctx, state)
				require.NoError(t, err)
			}

			for _, attempt := range tt.attempts {
				consumed, err := repo.Consume(ctx, attempt.stateHash, attempt.provider, attempt.purpose, now)
				require.NoError(t, err)
				if attempt.wantNil {
					assert.Nil(t, consumed)
					continue
				}

				require.NotNil(t, consumed)
				require.NotNil(t, consumed.ConsumedAt)
				assert.Equal(t, attempt.stateHash, consumed.StateHash)
			}
		})
	}
}

func TestBunFlowStateRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		states       []*types.OAuthFlowState
		wantDeleted  int64
		wantRetained []string
		wantRemoved  []string
	}{
		{
			name: "deletes only expired states",
			states: []*types.OAuthFlowState{
				createTestFlowState("expired", testNow().Add(-time.Minute)),
				createTestFlowState("valid", testNow().Add(time.Minute)),
			},
			wantDeleted:  1,
			wantRetained: []string{"valid"},
			wantRemoved:  []string{"expired"},
		},
		{
			name: "keeps all states when none are expired",
			states: []*types.OAuthFlowState{
				createTestFlowState("valid-1", testNow().Add(time.Minute)),
				createTestFlowState("valid-2", testNow().Add(time.Hour)),
			},
			wantDeleted:  0,
			wantRetained: []string{"valid-1", "valid-2"},
		},
		{
			name: "deletes all expired states",
			states: []*types.OAuthFlowState{
				createTestFlowState("expired-1", testNow().Add(-time.Hour)),
				createTestFlowState("expired-2", testNow().Add(-time.Minute)),
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
			repo := NewBunFlowStateRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, state := range tt.states {
				_, err := repo.Create(ctx, state)
				require.NoError(t, err)
			}

			deleted, err := repo.DeleteExpired(ctx, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDeleted, deleted)

			for _, stateHash := range tt.wantRetained {
				state, err := repo.GetByStateHash(ctx, stateHash)
				require.NoError(t, err)
				assert.NotNil(t, state, "state %s should be retained", stateHash)
			}
			for _, stateHash := range tt.wantRemoved {
				state, err := repo.GetByStateHash(ctx, stateHash)
				require.NoError(t, err)
				assert.Nil(t, state, "state %s should be deleted", stateHash)
			}
		})
	}
}

func TestBunFlowStateRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rollback  bool
		wantStore bool
	}{
		{name: "commit persists the state", wantStore: true},
		{name: "rollback discards the state", rollback: true, wantStore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			repo := NewBunFlowStateRepository(db)
			state := createTestFlowState("state-hash", testNow().Add(5*time.Minute))
			rollback := errors.New("rollback")

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := repo.WithTx(tx).Create(ctx, state); err != nil {
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

			stored, err := repo.GetByStateHash(ctx, state.StateHash)
			require.NoError(t, err)
			if tt.wantStore {
				require.NotNil(t, stored)
				assert.Equal(t, state.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}
