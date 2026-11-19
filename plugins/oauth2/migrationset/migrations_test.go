package migrationset

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func newMigrationTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(dsn))
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func oauth2Tables() []string {
	return []string{
		"oauth_clients",
		"oauth_auth_requests",
		"oauth_consent_requests",
		"oauth_consents",
		"oauth_access_grants",
		"oauth_refresh_tokens",
		"oauth_flow_states",
	}
}

func TestSQLiteInitialMigrationCreatesOAuth2Schema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newMigrationTestDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)

	set := migrations.MigrationSet{PluginID: "oauth2", Migrations: ForProvider("sqlite")}
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

	for _, name := range oauth2Tables() {
		var count int
		err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(name)).Scan(ctx, &count)
		assert.NoError(t, err, "table %s should be queryable", name)
	}

	for _, retired := range []string{"authorization_codes", "oauth_login_transactions"} {
		var count int
		err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(retired)).Scan(ctx, &count)
		assert.Error(t, err, "retired table %s should not exist", retired)
	}
}

func TestSQLiteInitialMigrationAddsClientFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newMigrationTestDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{{PluginID: "oauth2", Migrations: ForProvider("sqlite")}}))

	client := createMigrationTestClient(t, "client-1")
	_, err = db.NewInsert().Model(client).Exec(ctx)
	require.NoError(t, err)

	stored := new(types.OAuthClient)
	require.NoError(t, db.NewSelect().Model(stored).Where("client_id = ?", "client-1").Scan(ctx))
	assert.Equal(t, types.OAuthApplicationTypeWeb, stored.ApplicationType)
	assert.Equal(t, types.OAuthAuthMethodClientSecretBasic, stored.AuthMethod)
	assert.False(t, stored.TrustedFirstParty)
	assert.True(t, stored.RequireConsent)
	assert.Equal(t, types.OAuthAccessTokenTypeJWT, stored.AccessTokenType)
	assert.Equal(t, 900, stored.AccessTokenLifetime)
	assert.Equal(t, 300, stored.IDTokenLifetime)
	assert.Equal(t, []string{"https://app.example.com/logout"}, stored.PostLogoutRedirectURIs)
	assert.Equal(t, []string{"https://api.example.com"}, stored.AllowedAudiences)
}

func TestSQLiteInitialMigrationEnforcesSingleActiveConsent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newMigrationTestDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)

	set := migrations.MigrationSet{PluginID: "oauth2", Migrations: ForProvider("sqlite")}
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

	insert := func(id string) error {
		_, err := db.NewInsert().Model(&types.OAuthConsent{
			ID:       id,
			UserID:   "user-1",
			ClientID: "client-1",
			Scope:    "openid",
			Source:   types.OAuthConsentSourceExplicit,
		}).Exec(ctx)
		return err
	}

	require.NoError(t, insert("consent-1"))
	require.Error(t, insert("consent-2"))

	_, err = db.NewUpdate().
		Model((*types.OAuthConsent)(nil)).
		Set("revoked_at = CURRENT_TIMESTAMP").
		Where("id = ?", "consent-1").
		Exec(ctx)
	require.NoError(t, err)

	require.NoError(t, insert("consent-2"))
}

func TestSQLiteInitialMigrationRollsBack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newMigrationTestDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)

	set := migrations.MigrationSet{PluginID: "oauth2", Migrations: ForProvider("sqlite")}
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))
	require.NoError(t, migrator.RollbackAll(ctx, []migrations.MigrationSet{set}))

	for _, name := range oauth2Tables() {
		var count int
		err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(name)).Scan(ctx, &count)
		assert.Error(t, err, "table %s should be dropped", name)
	}
}

func TestMigrationsRunForSupportedProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider string
		db       func(*testing.T) *bun.DB
	}{
		{provider: "sqlite", db: newMigrationTestDB},
		{provider: "postgres", db: internaltests.NewPostgresIntegrationDB},
		{provider: "mysql", db: internaltests.NewMySQLIntegrationDB},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Parallel()

			db := tt.db(t)
			if db == nil {
				return
			}

			ctx := context.Background()
			migrator, err := migrations.NewMigrator(db, nil)
			require.NoError(t, err)

			set := migrations.MigrationSet{PluginID: "oauth2", Migrations: ForProvider(tt.provider)}
			require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

			for _, name := range oauth2Tables() {
				var count int
				err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(name)).Scan(ctx, &count)
				assert.NoError(t, err, "table %s should be queryable", name)
			}

			require.NoError(t, migrator.RollbackAll(ctx, []migrations.MigrationSet{set}))
		})
	}
}

func createMigrationTestClient(t *testing.T, clientID string) *types.OAuthClient {
	t.Helper()
	return &types.OAuthClient{
		ID:                     "client-" + clientID,
		ClientID:               clientID,
		ClientSecretHash:       "hash",
		ClientSecretSalt:       "salt",
		ClientSecretPrefix:     "authula",
		Name:                   "client",
		RedirectURIs:           []string{"https://app.example.com/callback"},
		PostLogoutRedirectURIs: []string{"https://app.example.com/logout"},
		AllowedGrants:          []string{types.OAuthGrantTypeAuthorizationCode},
		AllowedScopes:          []string{types.ScopeOpenID},
		AllowedAudiences:       []string{"https://api.example.com"},
		OwnerType:              types.OAuthClientOwnerTypeUser,
		OwnerID:                "user-1",
		CreatorID:              "user-1",
		ApplicationType:        types.OAuthApplicationTypeWeb,
		AuthMethod:             types.OAuthAuthMethodClientSecretBasic,
		RequireConsent:         true,
		AccessTokenType:        types.OAuthAccessTokenTypeJWT,
		AccessTokenLifetime:    900,
		IDTokenLifetime:        300,
		IsActive:               true,
	}
}
