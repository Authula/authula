package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
)

func TestCoreMigrationsRunForSupportedProviders(t *testing.T) {
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
			set, err := migrations.CoreMigrationSet(tt.provider)
			require.NoError(t, err)

			migrator, err := migrations.NewMigrator(db, nil)
			require.NoError(t, err)
			require.NoError(t, migrator.Migrate(ctx, []migrations.MigrationSet{set}))

			var keyCount int
			require.Error(t, db.NewRaw("SELECT COUNT(*) FROM jwks").Scan(ctx, &keyCount), "core must not create the jwks table")

			require.NoError(t, migrator.RollbackAll(ctx, []migrations.MigrationSet{set}))
		})
	}
}
