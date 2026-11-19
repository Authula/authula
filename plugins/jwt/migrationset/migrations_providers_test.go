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

func TestJWTMigrationsRunForSupportedProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider string
		db       func(*testing.T) *bun.DB
	}{
		{provider: "sqlite", db: internaltests.NewSQLiteIntegrationDB},
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
			coreSet, err := migrations.CoreMigrationSet(tt.provider)
			require.NoError(t, err)
			set := migrations.MigrationSet{
				PluginID:   testPluginID,
				Migrations: JWTMigrationsForProvider(tt.provider),
			}

			migrator, err := migrations.NewMigrator(db, nil)
			require.NoError(t, err)
			require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, set}))

			for _, table := range []string{"refresh_tokens", "jwks"} {
				var count int
				require.NoErrorf(t, db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(table)).Scan(ctx, &count), "table %s should exist", table)
			}

			require.NoError(t, migrator.RollbackAll(ctx, []migrations.MigrationSet{set}))

			for _, table := range []string{"refresh_tokens", "jwks"} {
				var count int
				assert.Errorf(t, db.NewRaw("SELECT COUNT(*) FROM ?", bun.Ident(table)).Scan(ctx, &count), "table %s should be dropped", table)
			}
		})
	}
}
