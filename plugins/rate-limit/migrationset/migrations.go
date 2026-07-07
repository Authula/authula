package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func Migrations() []migrations.Migration {
	return []migrations.Migration{postgresInitial(), postgresRules()}
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
