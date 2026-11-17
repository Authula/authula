package secondarystorage

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
	testSecondaryStoragePluginID = "secondary_storage"
	secondaryStorageInitialVer   = "20260129000000_secondary_storage_initial"
)

func TestSecondaryStorageMigrationsForProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
		want     []string
	}{
		{name: "sqlite", provider: "sqlite", want: []string{secondaryStorageInitialVer}},
		{name: "postgres", provider: "postgres", want: []string{secondaryStorageInitialVer}},
		{name: "mysql", provider: "mysql", want: []string{secondaryStorageInitialVer}},
		{name: "normalizes case and whitespace", provider: "  MySQL ", want: []string{secondaryStorageInitialVer}},
		{name: "unknown provider returns nothing", provider: "oracle", want: nil},
		{name: "empty provider returns nothing", provider: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := secondaryStorageMigrationsForProvider(tt.provider)

			require.Len(t, got, len(tt.want))
			for i, version := range tt.want {
				assert.Equal(t, version, got[i].Version)
				assert.NotNil(t, got[i].Up)
				assert.NotNil(t, got[i].Down)
			}
		})
	}
}

func TestSecondaryStorageSQLiteMigrationCreatesAndDropsSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)

	set := migrations.MigrationSet{
		PluginID:   testSecondaryStoragePluginID,
		Migrations: secondaryStorageMigrationsForProvider("sqlite"),
	}

	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))
	assertKeyValueStoreTableExists(t, db)
	assertKeyValueStoreIndexExists(t, db)

	applied, err := migrator.ListApplied(ctx, testSecondaryStoragePluginID)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, secondaryStorageInitialVer, applied[0].Version)

	// Re-applying an already applied migration must be a no-op.
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

	require.NoError(t, migrator.RollbackAll(ctx, []migrations.MigrationSet{set}))
	assertKeyValueStoreTableMissing(t, db)

	applied, err = migrator.ListApplied(ctx, testSecondaryStoragePluginID)
	require.NoError(t, err)
	assert.Empty(t, applied)
}

// TestSecondaryStorageDialectVariantsRunAgainstSQLite executes the postgres and
// mysql migration functions so their statements are exercised. The
// dialect-specific DDL cannot fully apply to SQLite, which is exactly what the
// expected errors capture.
func TestSecondaryStorageDialectVariantsRunAgainstSQLite(t *testing.T) {
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

			for _, migration := range secondaryStorageMigrationsForProvider(tt.provider) {
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

func TestSecondaryStoragePluginMigrationsMatchMigrationSet(t *testing.T) {
	t.Parallel()

	plugin := New(SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderDatabase})

	for _, provider := range []string{"sqlite", "postgres", "mysql"} {
		got := plugin.Migrations(provider)
		want := secondaryStorageMigrationsForProvider(provider)

		require.Len(t, got, len(want))
		for i := range want {
			assert.Equal(t, want[i].Version, got[i].Version, "provider %s", provider)
			assert.NotNil(t, got[i].Up)
			assert.NotNil(t, got[i].Down)
		}
	}
}

func assertKeyValueStoreTableExists(t *testing.T, db *bun.DB) {
	t.Helper()

	var count int
	err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident("key_value_store")).
		Scan(context.Background(), &count)
	assert.NoError(t, err, "key_value_store table should be queryable")
}

func assertKeyValueStoreTableMissing(t *testing.T, db *bun.DB) {
	t.Helper()

	var count int
	err := db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident("key_value_store")).
		Scan(context.Background(), &count)
	assert.Error(t, err, "key_value_store table should be dropped")
}

func assertKeyValueStoreIndexExists(t *testing.T, db *bun.DB) {
	t.Helper()

	var count int
	err := db.NewRaw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?",
		"idx_key_value_store_expires_at",
	).Scan(context.Background(), &count)

	require.NoError(t, err)
	assert.Equal(t, 1, count, "idx_key_value_store_expires_at index should exist")
}
