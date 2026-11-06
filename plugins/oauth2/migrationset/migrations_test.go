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

func TestSQLiteInitialMigrationCreatesOAuth2Schema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newMigrationTestDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)

	set := migrations.MigrationSet{PluginID: "oauth2", Migrations: ForProvider("sqlite")}
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

	for name := range map[string]any{
		"oauth_clients":            (*types.OAuthClient)(nil),
		"authorization_codes":      (*types.OAuthAuthorizationCode)(nil),
		"oauth_consent_requests":   (*types.OAuthConsentRequest)(nil),
		"oauth_consents":           (*types.OAuthConsent)(nil),
		"oauth_login_transactions": (*types.OAuthLoginTransaction)(nil),
		"oauth_flow_states":        (*types.OAuthFlowState)(nil),
	} {
		var count int
		err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(name)).Scan(ctx, &count)
		assert.NoError(t, err, "table %s should be queryable", name)
	}
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

	for name := range map[string]any{
		"oauth_clients":       (*types.OAuthClient)(nil),
		"authorization_codes": (*types.OAuthAuthorizationCode)(nil),
	} {
		var count int
		err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(name)).Scan(ctx, &count)
		assert.Error(t, err, "table %s should be dropped", name)
	}
}
