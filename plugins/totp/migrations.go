package totp

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
)

// MigrationSet returns the TOTP plugin migrations as a migration set
// compatible with the shared migrator.
func MigrationSet() migrations.MigrationSet {
	return migrations.MigrationSet{
		PluginID:   models.PluginTOTP.String(),
		Migrations: totpMigrations(),
	}
}

func totpMigrations() []migrations.Migration {
	return []migrations.Migration{
		{
			Version: "20260318000000_totp_initial",
			Up: func(ctx context.Context, tx bun.Tx) error {
				return migrations.ExecStatements(
					ctx,
					tx,
					`CREATE OR REPLACE FUNCTION totp_update_updated_at_func()
					RETURNS TRIGGER AS $$
					BEGIN
						NEW.updated_at = NOW();
						RETURN NEW;
					END;
				$$ LANGUAGE plpgsql;`,
					`CREATE TABLE IF NOT EXISTS totp (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					secret TEXT NOT NULL,
					backup_codes TEXT NOT NULL,
					enabled BOOLEAN NOT NULL DEFAULT FALSE,
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);`,
					`CREATE UNIQUE INDEX IF NOT EXISTS idx_totp_user_id ON totp(user_id);`,
					`DROP TRIGGER IF EXISTS totp_update_updated_at_trigger ON totp;`,
					`CREATE TRIGGER totp_update_updated_at_trigger
					BEFORE UPDATE ON totp
					FOR EACH ROW
					EXECUTE FUNCTION totp_update_updated_at_func();`,
					`CREATE TABLE IF NOT EXISTS trusted_devices (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					token VARCHAR(64) NOT NULL,
					user_agent TEXT NOT NULL DEFAULT '',
					expires_at TIMESTAMPTZ NOT NULL,
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);`,
					`CREATE INDEX IF NOT EXISTS idx_trusted_devices_user_id ON trusted_devices(user_id);`,
					`CREATE INDEX IF NOT EXISTS idx_trusted_devices_token ON trusted_devices(token);`,
					`CREATE INDEX IF NOT EXISTS idx_trusted_devices_expires_at ON trusted_devices(expires_at);`,
				)
			},
			Down: func(ctx context.Context, tx bun.Tx) error {
				return migrations.ExecStatements(
					ctx,
					tx,
					`DROP TRIGGER IF EXISTS totp_update_updated_at_trigger ON totp;`,
					`DROP FUNCTION IF EXISTS totp_update_updated_at_func();`,
					`DROP TABLE IF EXISTS trusted_devices;`,
					`DROP TABLE IF EXISTS totp;`,
				)
			},
		},
	}
}
