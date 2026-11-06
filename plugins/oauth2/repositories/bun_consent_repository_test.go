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

func assertConsentEqual(t *testing.T, want *types.OAuthConsent, got *types.OAuthConsent) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.ClientID, got.ClientID)
	assert.Equal(t, want.Scope, got.Scope)
	assert.Equal(t, want.Source, got.Source)
	assert.Equal(t, want.RevokedAt, got.RevokedAt)
	assert.True(t, want.CreatedAt.Equal(got.CreatedAt))
	assert.True(t, want.UpdatedAt.Equal(got.UpdatedAt))
}

func TestBunConsentRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []*types.OAuthConsent
		consent  *types.OAuthConsent
		wantErr  bool
	}{
		{
			name:    "creates and returns the consent",
			consent: createTestConsent("consent-1", "user-1", "client-1", "openid profile", testNow()),
		},
		{
			name: "rejects a duplicate consent id",
			existing: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			consent: createTestConsent("consent-1", "user-2", "client-2", "openid email", testNow()),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))

			for _, consent := range tt.existing {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}

			created, err := repo.Create(ctx, tt.consent)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertConsentEqual(t, tt.consent, created)

			stored, err := repo.GetByID(ctx, tt.consent.ID)
			require.NoError(t, err)
			assertConsentEqual(t, tt.consent, stored)
		})
	}
}

func TestBunConsentRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		consents  []*types.OAuthConsent
		revokeID  string
		consentID string
		wantFound bool
	}{
		{
			name: "returns an active consent",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid profile", testNow()),
				createTestConsent("consent-2", "user-2", "client-2", "openid email", testNow()),
			},
			consentID: "consent-2",
			wantFound: true,
		},
		{
			name: "returns a revoked consent",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			revokeID:  "consent-1",
			consentID: "consent-1",
			wantFound: true,
		},
		{
			name:      "returns nil for a missing consent",
			consentID: "missing",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, consent := range tt.consents {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			if tt.revokeID != "" {
				revoked, err := repo.Revoke(ctx, tt.revokeID, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			found, err := repo.GetByID(ctx, tt.consentID)
			require.NoError(t, err)
			if !tt.wantFound {
				assert.Nil(t, found)
				return
			}

			require.NotNil(t, found)
			assert.Equal(t, tt.consentID, found.ID)
			if tt.revokeID != "" {
				assert.NotNil(t, found.RevokedAt)
			} else {
				assert.Nil(t, found.RevokedAt)
			}
		})
	}
}

func TestBunConsentRepository_GetActiveByUserAndClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		consents  []*types.OAuthConsent
		revokeIDs []string
		userID    string
		clientID  string
		wantID    string
	}{
		{
			name: "returns the active consent for a user and client",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid profile", testNow()),
				createTestConsent("consent-2", "user-1", "client-2", "openid email", testNow()),
			},
			userID:   "user-1",
			clientID: "client-1",
			wantID:   "consent-1",
		},
		{
			name: "does not match a consent for another client",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			userID:   "user-1",
			clientID: "client-2",
		},
		{
			name: "does not match a consent for another user",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-2", "client-1", "openid", testNow()),
			},
			userID:   "user-1",
			clientID: "client-1",
		},
		{
			name: "does not return a revoked consent",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			revokeIDs: []string{"consent-1"},
			userID:    "user-1",
			clientID:  "client-1",
		},
		{
			name:     "returns nil when the user has no consent",
			userID:   "user-1",
			clientID: "client-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, consent := range tt.consents {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			for _, id := range tt.revokeIDs {
				revoked, err := repo.Revoke(ctx, id, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			active, err := repo.GetActiveByUserAndClient(ctx, tt.userID, tt.clientID)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, active)
				return
			}

			require.NotNil(t, active)
			assert.Equal(t, tt.wantID, active.ID)
		})
	}
}

func TestBunConsentRepository_ListActiveByUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		consents  []*types.OAuthConsent
		revokeIDs []string
		userID    string
		wantIDs   []string
	}{
		{
			name: "returns only the user's active consents newest first",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow().Add(-2*time.Minute)),
				createTestConsent("consent-2", "user-1", "client-2", "openid profile", testNow().Add(-time.Minute)),
				createTestConsent("consent-3", "user-2", "client-3", "openid", testNow()),
			},
			userID:  "user-1",
			wantIDs: []string{"consent-2", "consent-1"},
		},
		{
			name: "excludes revoked consents",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
				createTestConsent("consent-2", "user-1", "client-2", "openid", testNow()),
			},
			revokeIDs: []string{"consent-1"},
			userID:    "user-1",
			wantIDs:   []string{"consent-2"},
		},
		{
			name:    "returns an empty list when the user has no consents",
			userID:  "user-1",
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, consent := range tt.consents {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			for _, id := range tt.revokeIDs {
				revoked, err := repo.Revoke(ctx, id, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			consents, err := repo.ListActiveByUser(ctx, tt.userID)
			require.NoError(t, err)
			require.Len(t, consents, len(tt.wantIDs))
			for i, wantID := range tt.wantIDs {
				assert.Equal(t, wantID, consents[i].ID)
			}
		})
	}
}

func TestBunConsentRepository_Revoke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		consents    []*types.OAuthConsent
		revokeID    string
		preRevoke   bool
		wantRevoked bool
		wantMissing bool
	}{
		{
			name: "revokes an active consent",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			revokeID:    "consent-1",
			wantRevoked: true,
		},
		{
			name: "rejects an already revoked consent",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			revokeID:    "consent-1",
			preRevoke:   true,
			wantRevoked: false,
		},
		{
			name:        "returns false for a missing consent",
			revokeID:    "missing",
			wantRevoked: false,
			wantMissing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, consent := range tt.consents {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			if tt.preRevoke {
				revoked, err := repo.Revoke(ctx, tt.revokeID, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			revoked, err := repo.Revoke(ctx, tt.revokeID, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRevoked, revoked)

			consent, err := repo.GetByID(ctx, tt.revokeID)
			require.NoError(t, err)
			if tt.wantMissing {
				assert.Nil(t, consent)
				return
			}

			require.NotNil(t, consent)
			assert.NotNil(t, consent.RevokedAt)
		})
	}
}

func TestBunConsentRepository_RevokeByUserAndClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		consents        []*types.OAuthConsent
		preRevokeIDs    []string
		userID          string
		clientID        string
		wantRevoked     int64
		wantRevokedIDs  []string
		wantActiveIDs   []string
		replacement     *types.OAuthConsent
		wantActiveID    string
		wantActiveScope string
	}{
		{
			name: "revokes the active consent for one user and client",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
				createTestConsent("consent-2", "user-2", "client-1", "openid", testNow()),
			},
			userID:         "user-1",
			clientID:       "client-1",
			wantRevoked:    1,
			wantRevokedIDs: []string{"consent-1"},
			wantActiveIDs:  []string{"consent-2"},
		},
		{
			name: "returns zero when no active consent matches",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-2", "openid", testNow()),
			},
			userID:        "user-1",
			clientID:      "client-1",
			wantRevoked:   0,
			wantActiveIDs: []string{"consent-1"},
		},
		{
			name: "does not count an already revoked consent",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			preRevokeIDs:   []string{"consent-1"},
			userID:         "user-1",
			clientID:       "client-1",
			wantRevoked:    0,
			wantRevokedIDs: []string{"consent-1"},
		},
		{
			name: "allows a replacement consent after a scope change",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
			},
			userID:          "user-1",
			clientID:        "client-1",
			wantRevoked:     1,
			wantRevokedIDs:  []string{"consent-1"},
			replacement:     createTestConsent("consent-2", "user-1", "client-1", "openid profile", testNow()),
			wantActiveID:    "consent-2",
			wantActiveScope: "openid profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, consent := range tt.consents {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			for _, id := range tt.preRevokeIDs {
				revoked, err := repo.Revoke(ctx, id, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			revoked, err := repo.RevokeByUserAndClient(ctx, tt.userID, tt.clientID, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRevoked, revoked)

			assertConsentRevocationState(t, ctx, repo, tt.wantRevokedIDs, tt.wantActiveIDs)

			if tt.replacement == nil {
				return
			}

			_, err = repo.Create(ctx, tt.replacement)
			require.NoError(t, err)

			active, err := repo.GetActiveByUserAndClient(ctx, tt.userID, tt.clientID)
			require.NoError(t, err)
			require.NotNil(t, active)
			assert.Equal(t, tt.wantActiveID, active.ID)
			assert.Equal(t, tt.wantActiveScope, active.Scope)
		})
	}
}

func TestBunConsentRepository_RevokeByClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		consents       []*types.OAuthConsent
		preRevokeIDs   []string
		clientID       string
		wantRevoked    int64
		wantRevokedIDs []string
		wantActiveIDs  []string
	}{
		{
			name: "revokes every active consent for a client",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
				createTestConsent("consent-2", "user-2", "client-1", "openid", testNow()),
				createTestConsent("consent-3", "user-3", "client-2", "openid", testNow()),
			},
			clientID:       "client-1",
			wantRevoked:    2,
			wantRevokedIDs: []string{"consent-1", "consent-2"},
			wantActiveIDs:  []string{"consent-3"},
		},
		{
			name: "returns zero when no consent matches",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-2", "openid", testNow()),
			},
			clientID:      "client-1",
			wantRevoked:   0,
			wantActiveIDs: []string{"consent-1"},
		},
		{
			name: "ignores already revoked consents",
			consents: []*types.OAuthConsent{
				createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()),
				createTestConsent("consent-2", "user-2", "client-1", "openid", testNow()),
			},
			preRevokeIDs:   []string{"consent-1"},
			clientID:       "client-1",
			wantRevoked:    1,
			wantRevokedIDs: []string{"consent-1", "consent-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()

			for _, consent := range tt.consents {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			for _, id := range tt.preRevokeIDs {
				revoked, err := repo.Revoke(ctx, id, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			revoked, err := repo.RevokeByClient(ctx, tt.clientID, now)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRevoked, revoked)

			assertConsentRevocationState(t, ctx, repo, tt.wantRevokedIDs, tt.wantActiveIDs)
		})
	}
}

func assertConsentRevocationState(t *testing.T, ctx context.Context, repo ConsentRepository, revokedIDs []string, activeIDs []string) {
	t.Helper()

	for _, id := range revokedIDs {
		consent, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, consent)
		assert.NotNil(t, consent.RevokedAt, "consent %s should be revoked", id)
	}
	for _, id := range activeIDs {
		consent, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, consent)
		assert.Nil(t, consent.RevokedAt, "consent %s should remain active", id)
	}
}

func TestBunConsentRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rollback  bool
		wantStore bool
	}{
		{name: "commit persists the consent", wantStore: true},
		{name: "rollback discards the consent", rollback: true, wantStore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			repo := NewBunConsentRepository(db)
			consent := createTestConsent("consent-1", "user-1", "client-1", "openid", testNow())
			rollback := errors.New("rollback")

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := repo.WithTx(tx).Create(ctx, consent); err != nil {
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

			stored, err := repo.GetByID(ctx, consent.ID)
			require.NoError(t, err)
			if tt.wantStore {
				require.NotNil(t, stored)
				assert.Equal(t, consent.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}
