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

func assertLoginTransactionEqual(t *testing.T, want *types.OAuthLoginTransaction, got *types.OAuthLoginTransaction) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.TransactionHash, got.TransactionHash)
	assert.Equal(t, want.ClientID, got.ClientID)
	assert.Equal(t, want.RedirectURI, got.RedirectURI)
	assert.Equal(t, want.Scope, got.Scope)
	assert.Equal(t, want.State, got.State)
	assert.Equal(t, want.Nonce, got.Nonce)
	assert.Equal(t, want.CodeChallenge, got.CodeChallenge)
	assert.Equal(t, want.CodeChallengeMethod, got.CodeChallengeMethod)
	assert.Equal(t, want.Prompt, got.Prompt)
	assert.Equal(t, want.MaxAge, got.MaxAge)
	assert.True(t, want.AuthTime.Equal(got.AuthTime))
	assert.Equal(t, want.CompletedAt, got.CompletedAt)
	assert.True(t, want.ExpiresAt.Equal(got.ExpiresAt))
	assert.Equal(t, want.ConsumedAt, got.ConsumedAt)
}

func TestBunLoginTransactionRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		existing    []*types.OAuthLoginTransaction
		transaction *types.OAuthLoginTransaction
		wantErr     bool
	}{
		{
			name:        "creates and returns the transaction",
			transaction: createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
		},
		{
			name: "rejects a duplicate transaction hash",
			existing: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			},
			transaction: createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))

			for _, transaction := range tt.existing {
				_, err := repo.Create(ctx, transaction)
				require.NoError(t, err)
			}

			created, err := repo.Create(ctx, tt.transaction)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertLoginTransactionEqual(t, tt.transaction, created)

			stored, err := repo.GetByTransactionHash(ctx, tt.transaction.TransactionHash)
			require.NoError(t, err)
			assertLoginTransactionEqual(t, tt.transaction, stored)
		})
	}
}

func TestBunLoginTransactionRepository_GetByTransactionHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		transactions    []*types.OAuthLoginTransaction
		transactionHash string
		wantID          string
	}{
		{
			name: "returns the matching transaction",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash-1", testNow().Add(10*time.Minute)),
				createTestLoginTransaction("transaction-hash-2", testNow().Add(10*time.Minute)),
			},
			transactionHash: "transaction-hash-2",
			wantID:          "transaction-transaction-hash-2",
		},
		{
			name:            "returns nil for a missing transaction",
			transactionHash: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))

			for _, transaction := range tt.transactions {
				_, err := repo.Create(ctx, transaction)
				require.NoError(t, err)
			}

			found, err := repo.GetByTransactionHash(ctx, tt.transactionHash)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, found)
				return
			}

			require.NotNil(t, found)
			assert.Equal(t, tt.wantID, found.ID)
			assert.Equal(t, tt.transactionHash, found.TransactionHash)
		})
	}
}

func TestBunLoginTransactionRepository_MarkCompleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		transactions    []*types.OAuthLoginTransaction
		transactionHash string
		preComplete     bool
		preConsume      bool
		wantMarked      bool
		wantCompletedAt bool
		wantMissing     bool
	}{
		{
			name: "marks an active transaction completed",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			},
			transactionHash: "transaction-hash",
			wantMarked:      true,
			wantCompletedAt: true,
		},
		{
			name: "returns false when already completed",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			},
			transactionHash: "transaction-hash",
			preComplete:     true,
			wantMarked:      false,
			wantCompletedAt: true,
		},
		{
			name: "returns false when already consumed",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			},
			transactionHash: "transaction-hash",
			preConsume:      true,
			wantMarked:      false,
			wantCompletedAt: false,
		},
		{
			name: "returns false when expired",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("expired", testNow().Add(-time.Minute)),
			},
			transactionHash: "expired",
			wantMarked:      false,
			wantCompletedAt: false,
		},
		{
			name:            "returns false for a missing transaction",
			transactionHash: "missing",
			wantMarked:      false,
			wantMissing:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, transaction := range tt.transactions {
				_, err := repo.Create(ctx, transaction)
				require.NoError(t, err)
			}
			if tt.preComplete {
				completed, err := repo.MarkCompleted(ctx, tt.transactionHash, now)
				require.NoError(t, err)
				require.True(t, completed)
			}
			if tt.preConsume {
				consumed, err := repo.Consume(ctx, tt.transactionHash, now)
				require.NoError(t, err)
				require.NotNil(t, consumed)
			}

			marked, err := repo.MarkCompleted(ctx, tt.transactionHash, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMarked, marked)

			transaction, err := repo.GetByTransactionHash(ctx, tt.transactionHash)
			require.NoError(t, err)
			if tt.wantMissing {
				assert.Nil(t, transaction)
				return
			}

			require.NotNil(t, transaction)
			if tt.wantCompletedAt {
				assert.NotNil(t, transaction.CompletedAt)
			} else {
				assert.Nil(t, transaction.CompletedAt)
			}
		})
	}
}

func TestBunLoginTransactionRepository_Consume(t *testing.T) {
	t.Parallel()

	type consumeAttempt struct {
		transactionHash string
		wantNil         bool
	}

	tests := []struct {
		name         string
		transactions []*types.OAuthLoginTransaction
		preComplete  bool
		attempts     []consumeAttempt
	}{
		{
			name: "valid transaction can only be consumed once",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			},
			attempts: []consumeAttempt{
				{transactionHash: "transaction-hash"},
				{transactionHash: "transaction-hash", wantNil: true},
			},
		},
		{
			name: "consumes a completed transaction",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute)),
			},
			preComplete: true,
			attempts: []consumeAttempt{
				{transactionHash: "transaction-hash"},
			},
		},
		{
			name: "expired transaction is rejected",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("expired", testNow().Add(-time.Minute)),
			},
			attempts: []consumeAttempt{
				{transactionHash: "expired", wantNil: true},
			},
		},
		{
			name: "missing transaction is rejected",
			attempts: []consumeAttempt{
				{transactionHash: "missing", wantNil: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, transaction := range tt.transactions {
				_, err := repo.Create(ctx, transaction)
				require.NoError(t, err)
			}
			if tt.preComplete {
				completed, err := repo.MarkCompleted(ctx, tt.transactions[0].TransactionHash, now)
				require.NoError(t, err)
				require.True(t, completed)
			}

			for _, attempt := range tt.attempts {
				consumed, err := repo.Consume(ctx, attempt.transactionHash, now)
				require.NoError(t, err)
				if attempt.wantNil {
					assert.Nil(t, consumed)
					continue
				}

				require.NotNil(t, consumed)
				require.NotNil(t, consumed.ConsumedAt)
				assert.Equal(t, attempt.transactionHash, consumed.TransactionHash)
			}
		})
	}
}

func TestBunLoginTransactionRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		transactions []*types.OAuthLoginTransaction
		wantDeleted  int64
		wantRetained []string
		wantRemoved  []string
	}{
		{
			name: "deletes only expired transactions",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("expired", testNow().Add(-time.Minute)),
				createTestLoginTransaction("valid", testNow().Add(time.Minute)),
			},
			wantDeleted:  1,
			wantRetained: []string{"valid"},
			wantRemoved:  []string{"expired"},
		},
		{
			name: "keeps all transactions when none are expired",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("valid-1", testNow().Add(time.Minute)),
				createTestLoginTransaction("valid-2", testNow().Add(time.Hour)),
			},
			wantDeleted:  0,
			wantRetained: []string{"valid-1", "valid-2"},
		},
		{
			name: "deletes all expired transactions",
			transactions: []*types.OAuthLoginTransaction{
				createTestLoginTransaction("expired-1", testNow().Add(-time.Hour)),
				createTestLoginTransaction("expired-2", testNow().Add(-time.Minute)),
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
			repo := NewBunLoginTransactionRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, transaction := range tt.transactions {
				_, err := repo.Create(ctx, transaction)
				require.NoError(t, err)
			}

			deleted, err := repo.DeleteExpired(ctx, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDeleted, deleted)

			for _, transactionHash := range tt.wantRetained {
				transaction, err := repo.GetByTransactionHash(ctx, transactionHash)
				require.NoError(t, err)
				assert.NotNil(t, transaction, "transaction %s should be retained", transactionHash)
			}
			for _, transactionHash := range tt.wantRemoved {
				transaction, err := repo.GetByTransactionHash(ctx, transactionHash)
				require.NoError(t, err)
				assert.Nil(t, transaction, "transaction %s should be deleted", transactionHash)
			}
		})
	}
}

func TestBunLoginTransactionRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rollback  bool
		wantStore bool
	}{
		{name: "commit persists the transaction", wantStore: true},
		{name: "rollback discards the transaction", rollback: true, wantStore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			repo := NewBunLoginTransactionRepository(db)
			transaction := createTestLoginTransaction("transaction-hash", testNow().Add(10*time.Minute))
			rollback := errors.New("rollback")

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := repo.WithTx(tx).Create(ctx, transaction); err != nil {
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

			stored, err := repo.GetByTransactionHash(ctx, transaction.TransactionHash)
			require.NoError(t, err)
			if tt.wantStore {
				require.NotNil(t, stored)
				assert.Equal(t, transaction.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}
