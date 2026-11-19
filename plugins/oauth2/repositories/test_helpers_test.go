package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func newTestOAuth2DB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(dsn))
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	for _, model := range []any{
		(*types.OAuthClient)(nil),
		(*types.OAuthAuthRequest)(nil),
		(*types.OAuthConsentRequest)(nil),
		(*types.OAuthConsent)(nil),
		(*types.OAuthAccessGrant)(nil),
		(*types.OAuthRefreshToken)(nil),
		(*types.OAuthFlowState)(nil),
	} {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			t.Fatalf("failed to create oauth2 tables: %v", err)
		}
	}

	return db
}

func testNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func createTestClient(id string, clientID string, ownerType string, ownerID string, createdAt time.Time) *types.OAuthClient {
	return &types.OAuthClient{
		ID:                     id,
		ClientID:               clientID,
		ClientSecretHash:       "secret-hash",
		ClientSecretSalt:       "secret-salt",
		ClientSecretPrefix:     "authula_ab",
		Name:                   "client " + id,
		RedirectURIs:           []string{"https://app.example.com/callback"},
		PostLogoutRedirectURIs: []string{"https://app.example.com/logout/callback"},
		AllowedGrants:          []string{types.OAuthGrantTypeAuthorizationCode, types.OAuthGrantTypeRefreshToken},
		AllowedScopes:          []string{types.ScopeOpenID, types.ScopeProfile},
		AllowedAudiences:       []string{"https://api.example.com"},
		OwnerType:              ownerType,
		OwnerID:                ownerID,
		CreatorID:              ownerID,
		ApplicationType:        types.OAuthApplicationTypeWeb,
		AuthMethod:             types.OAuthAuthMethodClientSecretBasic,
		TrustedFirstParty:      false,
		RequireConsent:         true,
		AccessTokenType:        types.OAuthAccessTokenTypeJWT,
		AccessTokenLifetime:    900,
		IDTokenLifetime:        300,
		IsActive:               true,
		CreatedAt:              createdAt,
		UpdatedAt:              createdAt,
	}
}

func createTestAuthRequest(bindingHash string, clientID string, expiresAt time.Time) *types.OAuthAuthRequest {
	now := testNow()
	return &types.OAuthAuthRequest{
		ID:                  "request-" + bindingHash,
		ClientID:            clientID,
		UserID:              new("user-1"),
		SessionID:           new("session-1"),
		RedirectURI:         "https://app.example.com/callback",
		Scopes:              "openid profile",
		Audience:            new("https://api.example.com"),
		State:               new("state-1"),
		Nonce:               new("nonce-1"),
		ResponseType:        "code",
		ResponseMode:        new("query"),
		CodeChallenge:       "challenge",
		CodeChallengeMethod: types.CodeChallengeMethodS256,
		Prompt:              "consent",
		AuthTime:            &now,
		AMR:                 []string{"pwd"},
		ACR:                 new("urn:authula:acr:1"),
		BindingHash:         bindingHash,
		ConsentStatus:       types.OAuthConsentStatusPending,
		CreatedAt:           now,
		ExpiresAt:           expiresAt,
	}
}

func createTestConsentRequest(tokenHash string, userID string, clientID string, expiresAt time.Time) *types.OAuthConsentRequest {
	now := testNow()
	return &types.OAuthConsentRequest{
		ID:              "consent-request-" + tokenHash,
		TokenHash:       tokenHash,
		AuthRequestID:   "request-1",
		UserID:          userID,
		SessionID:       "session-1",
		ClientID:        clientID,
		RequestedScopes: "openid profile",
		Decision:        new(types.OAuthConsentDecisionAllow),
		GrantedScopes:   new("openid profile"),
		Origin:          new("https://app.example.com"),
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
	}
}

func createTestConsent(id string, userID string, clientID string, scope string, createdAt time.Time) *types.OAuthConsent {
	return &types.OAuthConsent{
		ID:        id,
		UserID:    userID,
		ClientID:  clientID,
		Scope:     scope,
		Source:    types.OAuthConsentSourceExplicit,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func createTestAccessGrant(jti string, familyID string, expiresAt time.Time) *types.OAuthAccessGrant {
	now := testNow()
	return &types.OAuthAccessGrant{
		ID:        "grant-" + jti,
		JTI:       jti,
		ClientID:  "client-1",
		UserID:    "user-1",
		SessionID: "session-1",
		FamilyID:  familyID,
		Scopes:    "openid profile",
		Audience:  new("https://api.example.com"),
		AuthTime:  &now,
		AMR:       []string{"pwd"},
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
}

func createTestRefreshToken(tokenHash string, familyID string, expiresAt time.Time) *types.OAuthRefreshToken {
	now := testNow()
	return &types.OAuthRefreshToken{
		ID:        "refresh-" + tokenHash,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		ClientID:  "client-1",
		UserID:    "user-1",
		SessionID: "session-1",
		Scopes:    "openid profile",
		Audience:  new("https://api.example.com"),
		AuthTime:  &now,
		AMR:       []string{"pwd"},
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
}

func createTestFlowState(stateHash string, expiresAt time.Time) *types.OAuthFlowState {
	return &types.OAuthFlowState{
		ID:                    "state-" + stateHash,
		StateHash:             stateHash,
		Provider:              "google",
		Purpose:               types.OAuthFlowPurposeLogin,
		CodeVerifierEncrypted: new("encrypted-verifier"),
		ExpiresAt:             expiresAt,
		CreatedAt:             testNow(),
	}
}
