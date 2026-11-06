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
		(*types.OAuthAuthorizationCode)(nil),
		(*types.OAuthConsentRequest)(nil),
		(*types.OAuthConsent)(nil),
		(*types.OAuthLoginTransaction)(nil),
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
		ID:                 id,
		ClientID:           clientID,
		ClientSecretHash:   "secret-hash",
		ClientSecretSalt:   "secret-salt",
		ClientSecretPrefix: "authula_ab",
		Name:               "client " + id,
		RedirectURIs:       []string{"https://app.example.com/callback"},
		AllowedGrants:      []string{types.OAuthGrantTypeAuthorizationCode},
		AllowedScopes:      []string{types.ScopeOpenID, types.ScopeProfile},
		OwnerType:          ownerType,
		OwnerID:            ownerID,
		CreatorID:          ownerID,
		IsActive:           true,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
	}
}

func createTestAuthorizationCode(codeHash string, clientID string, expiresAt time.Time) *types.OAuthAuthorizationCode {
	now := testNow()
	return &types.OAuthAuthorizationCode{
		ID:                  "code-" + codeHash,
		CodeHash:            codeHash,
		ClientID:            clientID,
		UserID:              "user-1",
		SessionID:           "session-1",
		ConsentID:           "consent-1",
		RedirectURI:         "https://app.example.com/callback",
		Scope:               "openid profile",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: types.CodeChallengeMethodS256,
		AuthTime:            now,
		ExpiresAt:           expiresAt,
		CreatedAt:           now,
	}
}

func createTestConsentRequest(requestHash string, userID string, clientID string, expiresAt time.Time) *types.OAuthConsentRequest {
	now := testNow()
	return &types.OAuthConsentRequest{
		ID:                  "request-" + requestHash,
		RequestHash:         requestHash,
		ClientID:            clientID,
		UserID:              userID,
		SessionID:           "session-1",
		RedirectURI:         "https://app.example.com/callback",
		Scope:               "openid profile",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: types.CodeChallengeMethodS256,
		Prompt:              "consent",
		AuthTime:            now,
		ExpiresAt:           expiresAt,
		CreatedAt:           now,
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

func createTestLoginTransaction(transactionHash string, expiresAt time.Time) *types.OAuthLoginTransaction {
	now := testNow()
	return &types.OAuthLoginTransaction{
		ID:                  "transaction-" + transactionHash,
		TransactionHash:     transactionHash,
		ClientID:            "client-1",
		RedirectURI:         "https://app.example.com/callback",
		Scope:               "openid",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: types.CodeChallengeMethodS256,
		Prompt:              "login",
		AuthTime:            now,
		ExpiresAt:           expiresAt,
		CreatedAt:           now,
	}
}

func createTestFlowState(stateHash string, expiresAt time.Time) *types.OAuthFlowState {
	return &types.OAuthFlowState{
		ID:                    "state-" + stateHash,
		StateHash:             stateHash,
		Provider:              "google",
		Purpose:               types.OAuthFlowPurposeAuthorization,
		CodeVerifierEncrypted: "encrypted-verifier",
		ExpiresAt:             expiresAt,
		CreatedAt:             testNow(),
	}
}
