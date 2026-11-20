package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func newClosedOAuth2DB(t *testing.T) *bun.DB {
	t.Helper()

	db := newTestOAuth2DB(t)
	require.NoError(t, db.Close())
	return db
}

func TestBunConsentRepository_UpdateScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing bool
		revoked  bool
		scope    string
		wantErr  error
		wantNil  bool
	}{
		{name: "updates an active consent", existing: true, scope: "openid profile email"},
		{name: "returns not found for a missing consent", scope: "openid", wantErr: ErrConsentNotFound, wantNil: true},
		{name: "returns not found for a revoked consent", existing: true, revoked: true, scope: "openid", wantErr: ErrConsentNotFound, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunConsentRepository(newTestOAuth2DB(t))
			now := testNow()
			consent := createTestConsent("consent-1", "user-1", "client-1", "openid", now)
			if tt.existing {
				_, err := repo.Create(ctx, consent)
				require.NoError(t, err)
			}
			if tt.revoked {
				revoked, err := repo.Revoke(ctx, consent.ID, now)
				require.NoError(t, err)
				require.True(t, revoked)
			}

			updated, err := repo.UpdateScope(ctx, "consent-1", tt.scope, now.Add(time.Minute))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, updated)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, tt.scope, updated.Scope)
			assert.True(t, updated.UpdatedAt.After(now))
		})
	}
}

func TestBunConsentRepository_UpdateScopeDatabaseError(t *testing.T) {
	t.Parallel()

	repo := NewBunConsentRepository(newClosedOAuth2DB(t))
	_, err := repo.UpdateScope(context.Background(), "consent-1", "openid", testNow())

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrConsentNotFound)
}

func TestBunAuthRequestRepository_GetByCodeHash(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunAuthRequestRepository(newTestOAuth2DB(t))
	request := createTestAuthRequest("binding-1", "client-1", testNow().Add(10*time.Minute))
	request.CodeHash = new("code-hash")
	_, err := repo.Create(ctx, request)
	require.NoError(t, err)

	found, err := repo.GetByCodeHash(ctx, "code-hash")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, request.ID, found.ID)

	missing, err := repo.GetByCodeHash(ctx, "missing")
	require.ErrorIs(t, err, ErrAuthRequestNotFound)
	assert.Nil(t, missing)
}

func TestBunRefreshTokenRepository_GetByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunRefreshTokenRepository(newTestOAuth2DB(t))
	token := createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Hour))
	_, err := repo.Create(ctx, token)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, token.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, token.ID, found.ID)

	missing, err := repo.GetByID(ctx, "missing")
	require.ErrorIs(t, err, ErrRefreshTokenNotFound)
	assert.Nil(t, missing)
}

func TestOAuth2Repositories_WithTx(t *testing.T) {
	t.Parallel()

	rollback := errors.New("rollback")

	tests := []struct {
		name       string
		store      func(context.Context, bun.Tx) error
		missing    func(context.Context, bun.IDB) error
		missingErr error
	}{
		{
			name: "auth request",
			store: func(ctx context.Context, tx bun.Tx) error {
				_, err := NewBunAuthRequestRepository(tx).WithTx(tx).Create(ctx, createTestAuthRequest("binding-1", "client-1", testNow().Add(time.Minute)))
				return err
			},
			missing: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAuthRequestRepository(db).GetByID(ctx, "request-binding-1")
				return err
			},
			missingErr: ErrAuthRequestNotFound,
		},
		{
			name: "consent request",
			store: func(ctx context.Context, tx bun.Tx) error {
				_, err := NewBunConsentRequestRepository(tx).WithTx(tx).Create(ctx, createTestConsentRequest("token-hash", "user-1", "client-1", testNow().Add(time.Minute)))
				return err
			},
			missing: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRequestRepository(db).GetByID(ctx, "consent-request-token-hash")
				return err
			},
			missingErr: ErrConsentRequestNotFound,
		},
		{
			name: "access grant",
			store: func(ctx context.Context, tx bun.Tx) error {
				_, err := NewBunAccessGrantRepository(tx).WithTx(tx).Create(ctx, createTestAccessGrant("jti-1", "family-1", testNow().Add(time.Minute)))
				return err
			},
			missing: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAccessGrantRepository(db).GetByID(ctx, "grant-jti-1")
				return err
			},
			missingErr: ErrAccessGrantNotFound,
		},
		{
			name: "refresh token",
			store: func(ctx context.Context, tx bun.Tx) error {
				_, err := NewBunRefreshTokenRepository(tx).WithTx(tx).Create(ctx, createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Minute)))
				return err
			},
			missing: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunRefreshTokenRepository(db).GetByID(ctx, "refresh-hash-1")
				return err
			},
			missingErr: ErrRefreshTokenNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				return tt.store(ctx, tx)
			})
			require.NoError(t, err)
			require.NoError(t, tt.missing(ctx, db))

			rollbackDB := newTestOAuth2DBNamed(t, t.Name()+"-rollback")
			err = rollbackDB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if err := tt.store(ctx, tx); err != nil {
					return err
				}
				return rollback
			})
			require.ErrorIs(t, err, rollback)
			require.ErrorIs(t, tt.missing(ctx, rollbackDB), tt.missingErr)
		})
	}
}

func TestOAuth2Repositories_DatabaseErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	token := createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Hour))
	replacement := createTestRefreshToken("hash-2", "family-1", testNow().Add(2*time.Hour))

	tests := []struct {
		name string
		call func(*bun.DB) error
	}{
		{
			name: "oauth client get by id",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).GetByID(ctx, "id-1")
				return err
			},
		},
		{
			name: "oauth client get by id including deleted",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).GetByIDIncludingDeleted(ctx, "id-1")
				return err
			},
		},
		{
			name: "oauth client get by client id",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).GetByClientID(ctx, "client-1")
				return err
			},
		},
		{
			name: "oauth client get by client id including deleted",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).GetByClientIDIncludingDeleted(ctx, "client-1")
				return err
			},
		},
		{
			name: "oauth client list",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).List(ctx, OAuthClientListFilter{})
				return err
			},
		},
		{
			name: "oauth client create",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
				return err
			},
		},
		{
			name: "oauth client update",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).Update(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()), nil)
				return err
			},
		},
		{
			name: "oauth client soft delete",
			call: func(db *bun.DB) error {
				_, err := NewBunOAuthClientRepository(db).SoftDelete(ctx, "id-1", testNow())
				return err
			},
		},
		{
			name: "auth request create",
			call: func(db *bun.DB) error {
				_, err := NewBunAuthRequestRepository(db).Create(ctx, createTestAuthRequest("binding-1", "client-1", testNow()))
				return err
			},
		},
		{
			name: "auth request get by id",
			call: func(db *bun.DB) error {
				_, err := NewBunAuthRequestRepository(db).GetByID(ctx, "request-1")
				return err
			},
		},
		{
			name: "auth request get by binding hash",
			call: func(db *bun.DB) error {
				_, err := NewBunAuthRequestRepository(db).GetByBindingHash(ctx, "binding-1")
				return err
			},
		},
		{
			name: "auth request get by code hash",
			call: func(db *bun.DB) error {
				_, err := NewBunAuthRequestRepository(db).GetByCodeHash(ctx, "code-hash")
				return err
			},
		},
		{
			name: "auth request consume",
			call: func(db *bun.DB) error {
				_, err := NewBunAuthRequestRepository(db).ConsumeCode(ctx, "code-hash", "client-1", testNow())
				return err
			},
		},
		{
			name: "auth request delete expired",
			call: func(db *bun.DB) error {
				_, err := NewBunAuthRequestRepository(db).DeleteExpired(ctx, testNow())
				return err
			},
		},
		{
			name: "consent request create",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRequestRepository(db).Create(ctx, createTestConsentRequest("token-hash", "user-1", "client-1", testNow()))
				return err
			},
		},
		{
			name: "consent request get by id",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRequestRepository(db).GetByID(ctx, "consent-request-1")
				return err
			},
		},
		{
			name: "consent request get by token hash",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRequestRepository(db).GetByTokenHash(ctx, "token-hash")
				return err
			},
		},
		{
			name: "consent request get by completion code hash",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRequestRepository(db).GetByCompletionCodeHash(ctx, "completion-hash")
				return err
			},
		},
		{
			name: "consent request consume",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRequestRepository(db).Consume(ctx, "token-hash", testNow())
				return err
			},
		},
		{
			name: "consent request delete expired",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRequestRepository(db).DeleteExpired(ctx, testNow())
				return err
			},
		},
		{
			name: "consent create",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()))
				return err
			},
		},
		{
			name: "consent get by id",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).GetByID(ctx, "consent-1")
				return err
			},
		},
		{
			name: "consent get active",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).GetActiveByUserAndClient(ctx, "user-1", "client-1")
				return err
			},
		},
		{
			name: "consent list active",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).ListActiveByUser(ctx, "user-1")
				return err
			},
		},
		{
			name: "consent update scope",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).UpdateScope(ctx, "consent-1", "openid", testNow())
				return err
			},
		},
		{
			name: "consent revoke",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).Revoke(ctx, "consent-1", testNow())
				return err
			},
		},
		{
			name: "consent revoke by user and client",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).RevokeByUserAndClient(ctx, "user-1", "client-1", testNow())
				return err
			},
		},
		{
			name: "consent revoke by client",
			call: func(db *bun.DB) error {
				_, err := NewBunConsentRepository(db).RevokeByClient(ctx, "client-1", testNow())
				return err
			},
		},
		{
			name: "access grant create",
			call: func(db *bun.DB) error {
				_, err := NewBunAccessGrantRepository(db).Create(ctx, createTestAccessGrant("jti-1", "family-1", testNow()))
				return err
			},
		},
		{
			name: "access grant get by id",
			call: func(db *bun.DB) error {
				_, err := NewBunAccessGrantRepository(db).GetByID(ctx, "grant-1")
				return err
			},
		},
		{
			name: "access grant get by jti",
			call: func(db *bun.DB) error {
				_, err := NewBunAccessGrantRepository(db).GetByJTI(ctx, "jti-1")
				return err
			},
		},
		{
			name: "access grant revoke",
			call: func(db *bun.DB) error {
				_, err := NewBunAccessGrantRepository(db).Revoke(ctx, "jti-1", testNow())
				return err
			},
		},
		{
			name: "access grant revoke family",
			call: func(db *bun.DB) error {
				_, err := NewBunAccessGrantRepository(db).RevokeFamily(ctx, "family-1", testNow())
				return err
			},
		},
		{
			name: "access grant delete expired",
			call: func(db *bun.DB) error {
				_, err := NewBunAccessGrantRepository(db).DeleteExpired(ctx, testNow())
				return err
			},
		},
		{
			name: "refresh token create",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).Create(ctx, token)
				return err
			},
		},
		{
			name: "refresh token get by id",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).GetByID(ctx, "refresh-1")
				return err
			},
		},
		{
			name: "refresh token get by token hash",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).GetByTokenHash(ctx, "hash-1")
				return err
			},
		},
		{
			name: "refresh token rotate",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).Rotate(ctx, "hash-1", replacement, testNow())
				return err
			},
		},
		{
			name: "refresh token rotate without replacement",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).Rotate(ctx, "hash-1", nil, testNow())
				return err
			},
		},
		{
			name: "refresh token revoke family",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).RevokeFamily(ctx, "family-1", testNow())
				return err
			},
		},
		{
			name: "refresh token delete expired",
			call: func(db *bun.DB) error {
				_, err := NewBunRefreshTokenRepository(db).DeleteExpired(ctx, testNow())
				return err
			},
		},
		{
			name: "flow state create",
			call: func(db *bun.DB) error {
				_, err := NewBunFlowStateRepository(db).Create(ctx, createTestFlowState("state-hash", testNow()))
				return err
			},
		},
		{
			name: "flow state get",
			call: func(db *bun.DB) error {
				_, err := NewBunFlowStateRepository(db).GetByStateHash(ctx, "state-hash")
				return err
			},
		},
		{
			name: "flow state consume",
			call: func(db *bun.DB) error {
				_, err := NewBunFlowStateRepository(db).Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeLogin, testNow())
				return err
			},
		},
		{
			name: "flow state delete expired",
			call: func(db *bun.DB) error {
				_, err := NewBunFlowStateRepository(db).DeleteExpired(ctx, testNow())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call(newClosedOAuth2DB(t))
			require.Error(t, err)
		})
	}
}

func TestOAuth2Repositories_MutationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		table   string
		trigger string
		seed    func(context.Context, bun.IDB)
		call    func(context.Context, bun.IDB) error
		wantMsg string
	}{
		{
			name:    "oauth client insert",
			table:   "oauth_clients",
			trigger: "BEFORE INSERT ON oauth_clients",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunOAuthClientRepository(db).Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
				return err
			},
			wantMsg: "insert oauth client",
		},
		{
			name:    "oauth client update",
			table:   "oauth_clients",
			trigger: "BEFORE UPDATE ON oauth_clients",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunOAuthClientRepository(db).Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunOAuthClientRepository(db).Update(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()), nil)
				return err
			},
			wantMsg: "update oauth client",
		},
		{
			name:    "auth request insert",
			table:   "oauth_auth_requests",
			trigger: "BEFORE INSERT ON oauth_auth_requests",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAuthRequestRepository(db).Create(ctx, createTestAuthRequest("binding-1", "client-1", testNow().Add(time.Minute)))
				return err
			},
			wantMsg: "insert oauth auth request",
		},
		{
			name:    "auth request consume",
			table:   "oauth_auth_requests",
			trigger: "BEFORE UPDATE ON oauth_auth_requests",
			seed: func(ctx context.Context, db bun.IDB) {
				request := createTestAuthRequest("binding-1", "client-1", testNow().Add(time.Minute))
				request.CodeHash = new("code-hash")
				request.CodeIssuedAt = new(testNow())
				_, err := NewBunAuthRequestRepository(db).Create(ctx, request)
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAuthRequestRepository(db).ConsumeCode(ctx, "code-hash", "client-1", testNow())
				return err
			},
			wantMsg: "consume oauth authorization code",
		},
		{
			name:    "consent request insert",
			table:   "oauth_consent_requests",
			trigger: "BEFORE INSERT ON oauth_consent_requests",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRequestRepository(db).Create(ctx, createTestConsentRequest("token-hash", "user-1", "client-1", testNow().Add(time.Minute)))
				return err
			},
			wantMsg: "insert oauth consent request",
		},
		{
			name:    "consent request consume",
			table:   "oauth_consent_requests",
			trigger: "BEFORE UPDATE ON oauth_consent_requests",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunConsentRequestRepository(db).Create(ctx, createTestConsentRequest("token-hash", "user-1", "client-1", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRequestRepository(db).Consume(ctx, "token-hash", testNow())
				return err
			},
			wantMsg: "consume oauth consent request",
		},
		{
			name:    "consent insert",
			table:   "oauth_consents",
			trigger: "BEFORE INSERT ON oauth_consents",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRepository(db).Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()))
				return err
			},
			wantMsg: "insert oauth consent",
		},
		{
			name:    "consent update scope",
			table:   "oauth_consents",
			trigger: "BEFORE UPDATE ON oauth_consents",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunConsentRepository(db).Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRepository(db).UpdateScope(ctx, "consent-1", "openid profile", testNow())
				return err
			},
			wantMsg: "update oauth consent scope",
		},
		{
			name:    "access grant insert",
			table:   "oauth_access_grants",
			trigger: "BEFORE INSERT ON oauth_access_grants",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAccessGrantRepository(db).Create(ctx, createTestAccessGrant("jti-1", "family-1", testNow().Add(time.Minute)))
				return err
			},
			wantMsg: "insert oauth access grant",
		},
		{
			name:    "access grant revoke",
			table:   "oauth_access_grants",
			trigger: "BEFORE UPDATE ON oauth_access_grants",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunAccessGrantRepository(db).Create(ctx, createTestAccessGrant("jti-1", "family-1", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAccessGrantRepository(db).Revoke(ctx, "jti-1", testNow())
				return err
			},
			wantMsg: "revoke oauth access grant",
		},
		{
			name:    "refresh token insert",
			table:   "oauth_refresh_tokens",
			trigger: "BEFORE INSERT ON oauth_refresh_tokens",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunRefreshTokenRepository(db).Create(ctx, createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Minute)))
				return err
			},
			wantMsg: "insert oauth refresh token",
		},
		{
			name:    "refresh token consume",
			table:   "oauth_refresh_tokens",
			trigger: "BEFORE UPDATE ON oauth_refresh_tokens",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunRefreshTokenRepository(db).Create(ctx, createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunRefreshTokenRepository(db).Rotate(ctx, "hash-1", createTestRefreshToken("hash-2", "family-1", testNow().Add(2*time.Minute)), testNow())
				return err
			},
			wantMsg: "consume oauth refresh token",
		},
		{
			name:    "refresh token replacement insert",
			table:   "oauth_refresh_tokens",
			trigger: "BEFORE INSERT ON oauth_refresh_tokens",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunRefreshTokenRepository(db).Create(ctx, createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunRefreshTokenRepository(db).Rotate(ctx, "hash-1", createTestRefreshToken("hash-2", "family-1", testNow().Add(2*time.Minute)), testNow())
				return err
			},
			wantMsg: "insert replacement oauth refresh token",
		},
		{
			name:    "flow state insert",
			table:   "oauth_flow_states",
			trigger: "BEFORE INSERT ON oauth_flow_states",
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunFlowStateRepository(db).Create(ctx, createTestFlowState("state-hash", testNow().Add(time.Minute)))
				return err
			},
			wantMsg: "insert oauth flow state",
		},
		{
			name:    "flow state consume",
			table:   "oauth_flow_states",
			trigger: "BEFORE UPDATE ON oauth_flow_states",
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunFlowStateRepository(db).Create(ctx, createTestFlowState("state-hash", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunFlowStateRepository(db).Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeLogin, testNow())
				return err
			},
			wantMsg: "consume oauth flow state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			if tt.seed != nil {
				tt.seed(ctx, db)
			}
			trigger := fmt.Sprintf("CREATE TRIGGER fail_%s %s BEGIN SELECT RAISE(ABORT, 'blocked'); END;", strings.ReplaceAll(tt.name, " ", "_"), tt.trigger)
			if _, err := db.NewRaw(trigger).Exec(ctx); err != nil {
				t.Fatalf("failed to create trigger: %v", err)
			}

			err := tt.call(ctx, db)
			require.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestOAuth2Repositories_ReloadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		trigger string
		seed    func(context.Context, bun.IDB)
		call    func(context.Context, bun.IDB) error
		wantMsg string
	}{
		{
			name: "oauth client create",
			trigger: `CREATE TRIGGER delete_oauth_client_after_insert AFTER INSERT ON oauth_clients
				BEGIN DELETE FROM oauth_clients WHERE id = NEW.id; END;`,
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunOAuthClientRepository(db).Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
				return err
			},
			wantMsg: "reload created oauth client",
		},
		{
			name: "oauth client update",
			trigger: `CREATE TRIGGER delete_oauth_client_after_update AFTER UPDATE ON oauth_clients
				BEGIN DELETE FROM oauth_clients WHERE id = NEW.id; END;`,
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunOAuthClientRepository(db).Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunOAuthClientRepository(db).Update(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()), nil)
				return err
			},
			wantMsg: "reload updated oauth client",
		},
		{
			name: "auth request consume",
			trigger: `CREATE TRIGGER delete_auth_request_after_update AFTER UPDATE ON oauth_auth_requests
				BEGIN DELETE FROM oauth_auth_requests WHERE id = NEW.id; END;`,
			seed: func(ctx context.Context, db bun.IDB) {
				request := createTestAuthRequest("binding-1", "client-1", testNow().Add(time.Minute))
				request.CodeHash = new("code-hash")
				request.CodeIssuedAt = new(testNow())
				_, err := NewBunAuthRequestRepository(db).Create(ctx, request)
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAuthRequestRepository(db).ConsumeCode(ctx, "code-hash", "client-1", testNow())
				return err
			},
			wantMsg: "reload consumed oauth auth request",
		},
		{
			name: "consent request consume",
			trigger: `CREATE TRIGGER delete_consent_request_after_update AFTER UPDATE ON oauth_consent_requests
				BEGIN DELETE FROM oauth_consent_requests WHERE id = NEW.id; END;`,
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunConsentRequestRepository(db).Create(ctx, createTestConsentRequest("token-hash", "user-1", "client-1", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRequestRepository(db).Consume(ctx, "token-hash", testNow())
				return err
			},
			wantMsg: "reload consumed oauth consent request",
		},
		{
			name: "consent update scope",
			trigger: `CREATE TRIGGER delete_consent_after_update AFTER UPDATE ON oauth_consents
				BEGIN DELETE FROM oauth_consents WHERE id = NEW.id; END;`,
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunConsentRepository(db).Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRepository(db).UpdateScope(ctx, "consent-1", "openid profile", testNow())
				return err
			},
			wantMsg: "reload updated oauth consent",
		},
		{
			name: "flow state consume",
			trigger: `CREATE TRIGGER delete_flow_state_after_update AFTER UPDATE ON oauth_flow_states
				BEGIN DELETE FROM oauth_flow_states WHERE id = NEW.id; END;`,
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunFlowStateRepository(db).Create(ctx, createTestFlowState("state-hash", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunFlowStateRepository(db).Consume(ctx, "state-hash", "google", types.OAuthFlowPurposeLogin, testNow())
				return err
			},
			wantMsg: "reload consumed oauth flow state",
		},
		{
			name: "refresh token rotate",
			trigger: `CREATE TRIGGER delete_refresh_token_after_update AFTER UPDATE ON oauth_refresh_tokens
				BEGIN DELETE FROM oauth_refresh_tokens WHERE id = NEW.id; END;`,
			seed: func(ctx context.Context, db bun.IDB) {
				_, err := NewBunRefreshTokenRepository(db).Create(ctx, createTestRefreshToken("hash-1", "family-1", testNow().Add(time.Minute)))
				require.NoError(t, err)
			},
			call: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunRefreshTokenRepository(db).Rotate(ctx, "hash-1", createTestRefreshToken("hash-2", "family-1", testNow().Add(2*time.Minute)), testNow())
				return err
			},
			wantMsg: "reload consumed oauth refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			if tt.seed != nil {
				tt.seed(ctx, db)
			}
			if _, err := db.NewRaw(tt.trigger).Exec(ctx); err != nil {
				t.Fatalf("failed to create trigger: %v", err)
			}

			err := tt.call(ctx, db)
			require.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestOAuth2Repositories_CreateReloadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		trigger string
		create  func(context.Context, bun.IDB) error
		wantMsg string
	}{
		{
			name: "auth request",
			trigger: `CREATE TRIGGER delete_auth_request_after_insert AFTER INSERT ON oauth_auth_requests
				BEGIN DELETE FROM oauth_auth_requests WHERE id = NEW.id; END;`,
			create: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAuthRequestRepository(db).Create(ctx, createTestAuthRequest("binding-1", "client-1", testNow()))
				return err
			},
			wantMsg: "reload created oauth auth request",
		},
		{
			name: "consent request",
			trigger: `CREATE TRIGGER delete_consent_request_after_insert AFTER INSERT ON oauth_consent_requests
				BEGIN DELETE FROM oauth_consent_requests WHERE id = NEW.id; END;`,
			create: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRequestRepository(db).Create(ctx, createTestConsentRequest("token-hash", "user-1", "client-1", testNow()))
				return err
			},
			wantMsg: "reload created oauth consent request",
		},
		{
			name: "consent",
			trigger: `CREATE TRIGGER delete_consent_after_insert AFTER INSERT ON oauth_consents
				BEGIN DELETE FROM oauth_consents WHERE id = NEW.id; END;`,
			create: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunConsentRepository(db).Create(ctx, createTestConsent("consent-1", "user-1", "client-1", "openid", testNow()))
				return err
			},
			wantMsg: "reload created oauth consent",
		},
		{
			name: "access grant",
			trigger: `CREATE TRIGGER delete_access_grant_after_insert AFTER INSERT ON oauth_access_grants
				BEGIN DELETE FROM oauth_access_grants WHERE id = NEW.id; END;`,
			create: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunAccessGrantRepository(db).Create(ctx, createTestAccessGrant("jti-1", "family-1", testNow()))
				return err
			},
			wantMsg: "reload created oauth access grant",
		},
		{
			name: "refresh token",
			trigger: `CREATE TRIGGER delete_refresh_token_after_insert AFTER INSERT ON oauth_refresh_tokens
				BEGIN DELETE FROM oauth_refresh_tokens WHERE id = NEW.id; END;`,
			create: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunRefreshTokenRepository(db).Create(ctx, createTestRefreshToken("hash-1", "family-1", testNow()))
				return err
			},
			wantMsg: "reload created oauth refresh token",
		},
		{
			name: "flow state",
			trigger: `CREATE TRIGGER delete_flow_state_after_insert AFTER INSERT ON oauth_flow_states
				BEGIN DELETE FROM oauth_flow_states WHERE id = NEW.id; END;`,
			create: func(ctx context.Context, db bun.IDB) error {
				_, err := NewBunFlowStateRepository(db).Create(ctx, createTestFlowState("state-hash", testNow()))
				return err
			},
			wantMsg: "reload created oauth flow state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newTestOAuth2DB(t)
			ctx := context.Background()
			if _, err := db.NewRaw(tt.trigger).Exec(ctx); err != nil {
				t.Fatalf("failed to create trigger: %v", err)
			}

			err := tt.create(ctx, db)
			require.ErrorContains(t, err, tt.wantMsg)
		})
	}
}
