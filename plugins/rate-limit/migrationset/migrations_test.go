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

const testPluginID = "rate-limit"

func TestForProvider(t *testing.T) {
	t.Parallel()

	wantVersions := []string{
		"20260130000000_rate_limit_initial",
		"20260130000001_rate_limit_rules",
	}

	tests := []struct {
		name     string
		provider string
		want     []string
	}{
		{name: "sqlite", provider: "sqlite", want: wantVersions},
		{name: "postgres", provider: "postgres", want: wantVersions},
		{name: "mysql", provider: "mysql", want: wantVersions},
		{name: "normalizes case and whitespace", provider: "  SqLiTe ", want: wantVersions},
		{name: "unknown provider returns nothing", provider: "oracle", want: nil},
		{name: "empty provider returns nothing", provider: " ", want: nil},
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
				if i > 0 {
					assert.Less(t, got[i-1].Version, got[i].Version, "versions should be ordered")
				}
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

	tables := []string{"rate_limits", "rate_limit_rules"}
	for _, table := range tables {
		assertTableExists(t, db, table)
	}

	applied, err := migrator.ListApplied(ctx, testPluginID)
	require.NoError(t, err)
	require.Len(t, applied, 2)
	assert.Equal(t, "20260130000000_rate_limit_initial", applied[0].Version)
	assert.Equal(t, "20260130000001_rate_limit_rules", applied[1].Version)

	// Re-applying already applied migrations must be a no-op.
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
// functions so their statements are exercised. SQLite accepts some of the
// dialect-specific DDL and rejects the rest, which the expectations capture.
func TestDialectVariantsRunAgainstSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name         string
		provider     string
		wantUpErrs   []bool
		wantDownErrs []bool
	}{
		{
			name:         "postgres",
			provider:     "postgres",
			wantUpErrs:   []bool{false, false},
			wantDownErrs: []bool{false, false},
		},
		{
			name:         "mysql",
			provider:     "mysql",
			wantUpErrs:   []bool{true, true},
			wantDownErrs: []bool{false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := internaltests.NewSQLiteIntegrationDB(t)
			providerMigrations := ForProvider(tt.provider)
			require.Len(t, providerMigrations, len(tt.wantUpErrs))

			for i, migration := range providerMigrations {
				upTx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				upErr := migration.Up(ctx, upTx)
				require.NoError(t, upTx.Rollback())
				assert.Equal(t, tt.wantUpErrs[i], upErr != nil, "migration %d up error: %v", i, upErr)

				downTx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				downErr := migration.Down(ctx, downTx)
				require.NoError(t, downTx.Rollback())
				assert.Equal(t, tt.wantDownErrs[i], downErr != nil, "migration %d down error: %v", i, downErr)
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
