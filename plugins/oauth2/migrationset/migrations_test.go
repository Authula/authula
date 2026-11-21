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

	var names []string
	require.NoError(t, db.NewRaw("SELECT name FROM pragma_table_info(?)", "oauth_clients").Scan(ctx, &names))
	assert.Contains(t, names, "client_secret_prefix")
	assert.Contains(t, names, "client_secret_start")
	assert.Contains(t, names, "client_secret_last")
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

// TestDialectVariantsRunAgainstSQLite executes the postgres and mysql migration
// functions so their statements are exercised. The dialect-specific DDL cannot
// fully apply to SQLite, which is exactly what the expected errors capture.
func TestDialectVariantsRunAgainstSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name        string
		provider    string
		wantUpErr   bool
		wantDownErr bool
	}{
		{name: "postgres", provider: "postgres", wantUpErr: true, wantDownErr: true},
		{name: "mysql", provider: "mysql", wantUpErr: true, wantDownErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := internaltests.NewSQLiteIntegrationDB(t)

			for _, migration := range ForProvider(tt.provider) {
				require.NotNil(t, migration.Up)
				require.NotNil(t, migration.Down)

				upTx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				upErr := migration.Up(ctx, upTx)
				require.NoError(t, upTx.Rollback())
				assert.Equal(t, tt.wantUpErr, upErr != nil, "up error: %v", upErr)

				downTx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				downErr := migration.Down(ctx, downTx)
				require.NoError(t, downTx.Rollback())
				assert.Equal(t, tt.wantDownErr, downErr != nil, "down error: %v", downErr)
			}
		})
	}
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
