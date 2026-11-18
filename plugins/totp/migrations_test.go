package totp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
)

func TestMigrationSet_ProviderSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
		want     int
	}{
		{name: "sqlite", provider: "sqlite", want: 1},
		{name: "postgres", provider: "postgres", want: 1},
		{name: "mysql", provider: "mysql", want: 1},
		{name: "provider is trimmed and lowercased", provider: " Postgres ", want: 1},
		{name: "empty provider", provider: "", want: 0},
		{name: "unsupported provider", provider: "oracle", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			set := MigrationSet(tt.provider)

			assert.Equal(t, models.PluginTOTP.String(), set.PluginID)
			require.Len(t, set.Migrations, tt.want)
			if tt.want == 0 {
				return
			}

			assert.Equal(t, "20260318000000_totp_initial", set.Migrations[0].Version)
			assert.NotNil(t, set.Migrations[0].Up)
			assert.NotNil(t, set.Migrations[0].Down)
		})
	}
}

func TestMigrationSet_SQLiteUpAndDown(t *testing.T) {
	t.Parallel()

	db := newEmptyTestDB(t)
	ctx := context.Background()

	migrator, err := migrations.NewMigrator(db, &tests.MockLogger{})
	require.NoError(t, err)
	coreSet, err := migrations.CoreMigrationSet("sqlite")
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{coreSet}))

	set := MigrationSet("sqlite")
	require.Len(t, set.Migrations, 1)
	migration := set.Migrations[0]

	require.False(t, tableExists(t, db, "totp"))
	require.False(t, tableExists(t, db, "trusted_devices"))

	require.NoError(t, db.RunInTx(ctx, nil, migration.Up))
	assert.True(t, tableExists(t, db, "totp"))
	assert.True(t, tableExists(t, db, "trusted_devices"))

	require.NoError(t, db.RunInTx(ctx, nil, migration.Down))
	assert.False(t, tableExists(t, db, "totp"))
	assert.False(t, tableExists(t, db, "trusted_devices"))
}

func tableExists(t *testing.T, db *bun.DB, table string) bool {
	t.Helper()

	var count int
	if err := db.NewRaw("SELECT COUNT(*) FROM "+table).Scan(context.Background(), &count); err != nil {
		require.ErrorContains(t, err, "no such table")
		return false
	}

	return true
}
