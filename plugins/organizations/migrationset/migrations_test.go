package migrationset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
)

const (
	testPluginID = "organizations"
	initialVer   = "20260409000000_organizations_initial"
)

func TestForProvider(t *testing.T) {
	t.Parallel()

	wantVersions := []string{initialVer}

	tests := []struct {
		name     string
		provider string
		want     []string
	}{
		{name: "sqlite", provider: "sqlite", want: wantVersions},
		{name: "postgres", provider: "postgres", want: wantVersions},
		{name: "mysql", provider: "mysql", want: wantVersions},
		{name: "normalizes case and whitespace", provider: " Postgres\n", want: wantVersions},
		{name: "unknown provider returns nothing", provider: "oracle", want: nil},
		{name: "empty provider returns nothing", provider: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ForProvider(tt.provider)

			require.Len(t, got, len(tt.want))
			for i, version := range tt.want {
				assert.Equal(t, version, got[i].Version)
				assert.NotNil(t, got[i].Up)
				assert.NotNil(t, got[i].Down)
			}
		})
	}
}

func TestSQLiteMigrationsCreateAndDropSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)

	set := migrations.MigrationSet{PluginID: testPluginID, Migrations: ForProvider("sqlite")}

	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

	tables := []string{
		"organizations",
		"organization_invitations",
		"organization_members",
		"organization_teams",
		"organization_team_members",
	}
	for _, table := range tables {
		assertTableExists(t, db, table)
	}

	for _, trigger := range []string{
		"update_organizations_updated_at_trigger",
		"update_organization_members_updated_at_trigger",
		"update_organization_teams_updated_at_trigger",
	} {
		assertTriggerExists(t, db, trigger)
	}

	applied, err := migrator.ListApplied(ctx, testPluginID)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, initialVer, applied[0].Version)

	// Re-applying an already applied migration must be a no-op.
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

	require.NoError(t, migrator.RollbackAll(ctx, []migrations.MigrationSet{set}))

	for _, table := range tables {
		assertTableMissing(t, db, table)
	}

	applied, err = migrator.ListApplied(ctx, testPluginID)
	require.NoError(t, err)
	assert.Empty(t, applied)
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

func assertTableExists(t *testing.T, db *bun.DB, table string) {
	t.Helper()

	var count int
	err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(table)).Scan(context.Background(), &count)
	assert.NoError(t, err, "table %s should be queryable", table)
}

func assertTableMissing(t *testing.T, db *bun.DB, table string) {
	t.Helper()

	var count int
	err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(table)).Scan(context.Background(), &count)
	assert.Error(t, err, "table %s should be dropped", table)
}

func assertTriggerExists(t *testing.T, db *bun.DB, trigger string) {
	t.Helper()

	var count int
	err := db.NewRaw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = ?", trigger).
		Scan(context.Background(), &count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "trigger %s should exist", trigger)
}
