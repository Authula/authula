package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func Migrations() []migrations.Migration {
	return []migrations.Migration{postgresInitial()}
}

func postgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260425000000_api_keys_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE OR REPLACE FUNCTION api_keys_set_updated_at_fn()
				RETURNS TRIGGER AS $$
				BEGIN
					NEW.updated_at = NOW();
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;`,
				`CREATE TABLE IF NOT EXISTS api_keys (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					key_hash VARCHAR(255) NOT NULL,
					name VARCHAR(255) NOT NULL,
					owner_type VARCHAR(255) NOT NULL,
					owner_id UUID NOT NULL,
					prefix VARCHAR(50),
					start VARCHAR(10) NOT NULL,
					last VARCHAR(10) NOT NULL,
					enabled BOOLEAN NOT NULL DEFAULT TRUE,
					rate_limit_enabled BOOLEAN NOT NULL DEFAULT FALSE,
					last_requested_at TIMESTAMP WITH TIME ZONE,
					expires_at TIMESTAMP WITH TIME ZONE,
					permissions JSONB,
					metadata JSONB,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_api_keys_owner_type CHECK (owner_type IN ('user', 'organization'))
				);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_owner_id ON api_keys(owner_id);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_owner_type_owner_id ON api_keys(owner_type, owner_id);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);`,
				`DROP TRIGGER IF EXISTS update_api_keys_updated_at_trigger ON api_keys;`,
				`CREATE TRIGGER update_api_keys_updated_at_trigger
				BEFORE UPDATE ON api_keys
				FOR EACH ROW
				EXECUTE FUNCTION api_keys_set_updated_at_fn();`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TRIGGER IF EXISTS update_api_keys_updated_at_trigger ON api_keys;`,
				`DROP TABLE IF EXISTS api_keys;`,
				`DROP FUNCTION IF EXISTS api_keys_set_updated_at_fn();`,
			)
		},
	}
}
