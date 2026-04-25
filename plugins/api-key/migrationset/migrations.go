package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func ForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{sqliteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{postgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{mysqlInitial()} },
	})
}

func sqliteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260425000000_api_keys_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`PRAGMA foreign_keys = ON;`,
				// -------------------------------
				`CREATE TABLE IF NOT EXISTS api_keys (
          id VARCHAR(255) NOT NULL PRIMARY KEY,
					key_hash VARCHAR(255) NOT NULL,
          name VARCHAR(255) NOT NULL,
          owner_type VARCHAR(255) NOT NULL,
          reference_id VARCHAR(255) NOT NULL,
          start VARCHAR(100) NOT NULL,
          prefix VARCHAR(255),
          enabled BOOLEAN NOT NULL DEFAULT 1,
          rate_limit_enabled BOOLEAN NOT NULL DEFAULT 0,
          rate_limit_time_window INTEGER,
          rate_limit_max_requests INTEGER,
          requests_remaining INTEGER,
          last_requested_at TIMESTAMP,
          expires_at TIMESTAMP,
          permissions TEXT,
          metadata TEXT,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_reference_id ON api_keys(reference_id);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_owner_type_reference_id ON api_keys(owner_type, reference_id);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);`,
				`DROP TRIGGER IF EXISTS api_keys_updated_at_trigger;`,
				`CREATE TRIGGER api_keys_updated_at_trigger 
        AFTER UPDATE ON api_keys
        BEGIN
          UPDATE api_keys SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;`,
				// -------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS api_keys;`,
			)
		},
	}
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
				// -------------------------------
				`CREATE TABLE IF NOT EXISTS api_keys (
				  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				  key_hash VARCHAR(255) NOT NULL,
				  name VARCHAR(255) NOT NULL,
				  owner_type VARCHAR(255) NOT NULL,
				  reference_id UUID NOT NULL,
				  start VARCHAR(255) NOT NULL,
				  prefix VARCHAR(255),
				  enabled BOOLEAN NOT NULL DEFAULT TRUE,
				  rate_limit_enabled BOOLEAN NOT NULL DEFAULT FALSE,
				  rate_limit_time_window INTEGER,
				  rate_limit_max_requests INTEGER,
				  requests_remaining INTEGER,
				  last_requested_at TIMESTAMP WITH TIME ZONE,
				  expires_at TIMESTAMP WITH TIME ZONE,
				  permissions JSONB,
				  metadata JSONB,
				  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				  CONSTRAINT chk_api_keys_owner_type CHECK (owner_type IN ('user', 'organization'))
				);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_reference_id ON api_keys(reference_id);`,
				`CREATE INDEX IF NOT EXISTS idx_api_keys_owner_type_reference_id ON api_keys(owner_type, reference_id);`,
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

func mysqlInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260425000000_api_keys_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS api_keys (
				  id BINARY(16) NOT NULL PRIMARY KEY,
				  key_hash VARCHAR(255) NOT NULL,
				  name VARCHAR(255) NOT NULL,
				  owner_type VARCHAR(255) NOT NULL,
				  reference_id BINARY(16) NOT NULL,
				  start VARCHAR(255) NOT NULL,
				  prefix VARCHAR(255) NULL,
				  enabled TINYINT(1) NOT NULL DEFAULT 1,
				  rate_limit_enabled TINYINT(1) NOT NULL DEFAULT 0,
				  rate_limit_time_window INT NULL,
				  rate_limit_max_requests INT NULL,
				  requests_remaining INT NULL,
				  last_requested_at TIMESTAMP NULL,
				  expires_at TIMESTAMP NULL,
				  permissions TEXT NULL,
				  metadata JSON NULL,
				  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				  CONSTRAINT chk_api_keys_owner_type CHECK (owner_type IN ('user', 'organization')),
				  UNIQUE KEY idx_api_keys_key_hash (key_hash),
				  INDEX idx_api_keys_reference_id (reference_id),
				  INDEX idx_api_keys_owner_type_reference_id (owner_type, reference_id),
				  INDEX idx_api_keys_expires_at (expires_at),
				  INDEX idx_api_keys_enabled (enabled)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS api_keys;`,
			)
		},
	}
}
