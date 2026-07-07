package secondarystorage

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func secondaryStorageMigrations() []migrations.Migration {
	return []migrations.Migration{
		{
			Version: "20260129000000_secondary_storage_initial",
			Up: func(ctx context.Context, tx bun.Tx) error {
				return migrations.ExecStatements(
					ctx,
					tx,
					`CREATE OR REPLACE FUNCTION key_value_store_update_updated_at_func()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;`,
					`CREATE TABLE IF NOT EXISTS key_value_store (
  key VARCHAR(255) PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);`,
					`CREATE INDEX IF NOT EXISTS idx_key_value_store_expires_at ON key_value_store(expires_at);`,
					`DROP TRIGGER IF EXISTS key_value_store_update_updated_at_trigger ON key_value_store;`,
					`CREATE TRIGGER key_value_store_update_updated_at_trigger
  BEFORE UPDATE ON key_value_store
  FOR EACH ROW
  EXECUTE FUNCTION key_value_store_update_updated_at_func();`,
				)
			},
			Down: func(ctx context.Context, tx bun.Tx) error {
				return migrations.ExecStatements(
					ctx,
					tx,
					`DROP TRIGGER IF EXISTS key_value_store_update_updated_at_trigger ON key_value_store;`,
					`DROP TABLE IF EXISTS key_value_store;`,
				)
			},
		},
	}
}
