package ratelimit

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
)

func rateLimitMigrationsForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{rateLimitSQLiteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{rateLimitPostgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{rateLimitMySQLInitial()} },
	})
}

func rateLimitSQLiteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000000_rate_limit_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`PRAGMA temp_store = MEMORY;`,
				`CREATE TEMP TABLE IF NOT EXISTS rate_limits (
  key TEXT PRIMARY KEY,
  count INTEGER NOT NULL,
  expires_at DATETIME NOT NULL
);`,
				`CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limits;`)
		},
	}
}

func rateLimitPostgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000000_rate_limit_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE UNLOGGED TABLE IF NOT EXISTS rate_limits (
  key VARCHAR(255) PRIMARY KEY,
  count INTEGER NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);`,
				`CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limits;`)
		},
	}
}

func rateLimitMySQLInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000000_rate_limit_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS rate_limits (
  key VARCHAR(255) PRIMARY KEY,
  count INTEGER NOT NULL,
  expires_at TIMESTAMP NOT NULL
) ENGINE=MEMORY;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limits;`)
		},
	}
}
