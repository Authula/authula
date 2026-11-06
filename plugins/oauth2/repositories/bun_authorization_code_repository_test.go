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

func assertAuthorizationCodeEqual(t *testing.T, want *types.OAuthAuthorizationCode, got *types.OAuthAuthorizationCode) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.CodeHash, got.CodeHash)
	assert.Equal(t, want.ClientID, got.ClientID)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.SessionID, got.SessionID)
	assert.Equal(t, want.ConsentID, got.ConsentID)
	assert.Equal(t, want.RedirectURI, got.RedirectURI)
	assert.Equal(t, want.Scope, got.Scope)
	assert.Equal(t, want.Nonce, got.Nonce)
	assert.Equal(t, want.CodeChallenge, got.CodeChallenge)
	assert.Equal(t, want.CodeChallengeMethod, got.CodeChallengeMethod)
	assert.True(t, want.AuthTime.Equal(got.AuthTime))
	assert.True(t, want.ExpiresAt.Equal(got.ExpiresAt))
	assert.Equal(t, want.ConsumedAt, got.ConsumedAt)
}

func TestBunAuthorizationCodeRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []*types.OAuthAuthorizationCode
		code     *types.OAuthAuthorizationCode
		wantErr  bool
	}{
		{
			name: "creates and returns the code",
			code: createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute)),
		},
		{
			name: "rejects a duplicate code hash",
			existing: []*types.OAuthAuthorizationCode{
				createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute)),
			},
			code:    createTestAuthorizationCode("hash-1", "client-2", testNow().Add(time.Minute)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))

			for _, code := range tt.existing {
				_, err := repo.Create(ctx, code)
				require.NoError(t, err)
			}

			created, err := repo.Create(ctx, tt.code)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertAuthorizationCodeEqual(t, tt.code, created)

			stored, err := repo.GetByCodeHash(ctx, tt.code.CodeHash)
			require.NoError(t, err)
			assertAuthorizationCodeEqual(t, tt.code, stored)
		})
	}
}

func TestBunAuthorizationCodeRepository_GetByCodeHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		codes     []*types.OAuthAuthorizationCode
		codeHash  string
		wantFound *types.OAuthAuthorizationCode
	}{
		{
			name: "returns the matching code",
			codes: []*types.OAuthAuthorizationCode{
				createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute)),
				createTestAuthorizationCode("hash-2", "client-2", testNow().Add(time.Minute)),
			},
			codeHash:  "hash-2",
			wantFound: createTestAuthorizationCode("hash-2", "client-2", testNow().Add(time.Minute)),
		},
		{
			name:      "returns nil for a missing code",
			codeHash:  "missing",
			wantFound: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))

			for _, code := range tt.codes {
				_, err := repo.Create(ctx, code)
				require.NoError(t, err)
			}

			found, err := repo.GetByCodeHash(ctx, tt.codeHash)
			require.NoError(t, err)
			if tt.wantFound == nil {
				assert.Nil(t, found)
				return
			}

			require.NotNil(t, found)
			assert.Equal(t, tt.wantFound.CodeHash, found.CodeHash)
			assert.Equal(t, tt.wantFound.ClientID, found.ClientID)
			assert.Equal(t, tt.wantFound.RedirectURI, found.RedirectURI)
		})
	}
}

func TestBunAuthorizationCodeRepository_Consume(t *testing.T) {
	t.Parallel()

	type consumeAttempt struct {
		codeHash string
		clientID string
		wantNil  bool
	}

	tests := []struct {
		name     string
		codes    []*types.OAuthAuthorizationCode
		attempts []consumeAttempt
	}{
		{
			name:  "valid code can only be consumed once",
			codes: []*types.OAuthAuthorizationCode{createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute))},
			attempts: []consumeAttempt{
				{codeHash: "hash-1", clientID: "client-1"},
				{codeHash: "hash-1", clientID: "client-1", wantNil: true},
			},
		},
		{
			name:  "wrong client does not consume the code",
			codes: []*types.OAuthAuthorizationCode{createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute))},
			attempts: []consumeAttempt{
				{codeHash: "hash-1", clientID: "client-2", wantNil: true},
				{codeHash: "hash-1", clientID: "client-1"},
			},
		},
		{
			name:  "empty client id skips the client filter",
			codes: []*types.OAuthAuthorizationCode{createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute))},
			attempts: []consumeAttempt{
				{codeHash: "hash-1", clientID: ""},
			},
		},
		{
			name:  "expired code is rejected",
			codes: []*types.OAuthAuthorizationCode{createTestAuthorizationCode("expired", "client-1", testNow().Add(-time.Minute))},
			attempts: []consumeAttempt{
				{codeHash: "expired", clientID: "client-1", wantNil: true},
			},
		},
		{
			name: "missing code is rejected",
			attempts: []consumeAttempt{
				{codeHash: "missing", clientID: "client-1", wantNil: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, code := range tt.codes {
				_, err := repo.Create(ctx, code)
				require.NoError(t, err)
			}

			for _, attempt := range tt.attempts {
				consumed, err := repo.Consume(ctx, attempt.codeHash, attempt.clientID, now)
				require.NoError(t, err)
				if attempt.wantNil {
					assert.Nil(t, consumed)
					continue
				}

				require.NotNil(t, consumed)
				require.NotNil(t, consumed.ConsumedAt)
				assert.Equal(t, attempt.codeHash, consumed.CodeHash)
			}
		})
	}
}

func TestBunAuthorizationCodeRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		codes        []*types.OAuthAuthorizationCode
		wantDeleted  int64
		wantRetained []string
		wantRemoved  []string
	}{
		{
			name: "deletes only expired codes",
			codes: []*types.OAuthAuthorizationCode{
				createTestAuthorizationCode("expired", "client-1", testNow().Add(-time.Minute)),
				createTestAuthorizationCode("valid", "client-1", testNow().Add(time.Minute)),
			},
			wantDeleted:  1,
			wantRetained: []string{"valid"},
			wantRemoved:  []string{"expired"},
		},
		{
			name: "keeps all codes when none are expired",
			codes: []*types.OAuthAuthorizationCode{
				createTestAuthorizationCode("valid-1", "client-1", testNow().Add(time.Minute)),
				createTestAuthorizationCode("valid-2", "client-1", testNow().Add(time.Hour)),
			},
			wantDeleted:  0,
			wantRetained: []string{"valid-1", "valid-2"},
		},
		{
			name: "deletes all expired codes",
			codes: []*types.OAuthAuthorizationCode{
				createTestAuthorizationCode("expired-1", "client-1", testNow().Add(-time.Hour)),
				createTestAuthorizationCode("expired-2", "client-1", testNow().Add(-time.Minute)),
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
			repo := NewBunAuthorizationCodeRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, code := range tt.codes {
				_, err := repo.Create(ctx, code)
				require.NoError(t, err)
			}

			deleted, err := repo.DeleteExpired(ctx, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDeleted, deleted)

			for _, codeHash := range tt.wantRetained {
				code, err := repo.GetByCodeHash(ctx, codeHash)
				require.NoError(t, err)
				assert.NotNil(t, code, "code %s should be retained", codeHash)
			}
			for _, codeHash := range tt.wantRemoved {
				code, err := repo.GetByCodeHash(ctx, codeHash)
				require.NoError(t, err)
				assert.Nil(t, code, "code %s should be deleted", codeHash)
			}
		})
	}
}

func TestBunAuthorizationCodeRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rollback  bool
		wantStore bool
	}{
		{name: "commit persists the code", wantStore: true},
		{name: "rollback discards the code", rollback: true, wantStore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			repo := NewBunAuthorizationCodeRepository(db)
			code := createTestAuthorizationCode("hash-1", "client-1", testNow().Add(time.Minute))
			rollback := errors.New("rollback")

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := repo.WithTx(tx).Create(ctx, code); err != nil {
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

			stored, err := repo.GetByCodeHash(ctx, code.CodeHash)
			require.NoError(t, err)
			if tt.wantStore {
				require.NotNil(t, stored)
				assert.Equal(t, code.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}
