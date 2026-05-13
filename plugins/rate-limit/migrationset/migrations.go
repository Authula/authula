package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func ForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{sqliteInitial(), sqliteRules()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{postgresInitial(), postgresRules()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{mysqlInitial(), mysqlRules()} },
	})
}

func sqliteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000000_rate_limit_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				// -----------------------------------
				`PRAGMA foreign_keys = ON;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS rate_limits (
					key TEXT NOT NULL PRIMARY KEY,
					count INTEGER NOT NULL,
					expires_at DATETIME NOT NULL
				);`,
				`CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);`,
				// -----------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limits;`)
		},
	}
}

func sqliteRules() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000001_rate_limit_rules",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS rate_limit_rules (
					key TEXT NOT NULL PRIMARY KEY,
					window_seconds INTEGER NOT NULL,
					max_requests INTEGER NOT NULL
				);`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limit_rules;`)
		},
	}
}

func postgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000000_rate_limit_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS rate_limits (
					key TEXT NOT NULL PRIMARY KEY,
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

func postgresRules() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000001_rate_limit_rules",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS rate_limit_rules (
					key TEXT NOT NULL PRIMARY KEY,
					window_seconds INTEGER NOT NULL,
					max_requests INTEGER NOT NULL
				);`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limit_rules;`)
		},
	}
}

func mysqlInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000000_rate_limit_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS rate_limits (
					key TEXT NOT NULL PRIMARY KEY,
					count INTEGER NOT NULL,
					expires_at TIMESTAMP NOT NULL
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limits;`)
		},
	}
}

func mysqlRules() migrations.Migration {
	return migrations.Migration{
		Version: "20260130000001_rate_limit_rules",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS rate_limit_rules (
					key TEXT NOT NULL PRIMARY KEY,
					window_seconds INTEGER NOT NULL,
					max_requests INTEGER NOT NULL
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS rate_limit_rules;`)
		},
	}
}
