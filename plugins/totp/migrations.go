package totp

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// MigrationSet returns the TOTP plugin migrations as a migration set
// compatible with the shared migrator.
func MigrationSet(provider string) migrations.MigrationSet {
	return migrations.MigrationSet{
		PluginID:   models.PluginTOTP.String(),
		Migrations: totpMigrationsForProvider(provider),
	}
}

func totpMigrationsForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{totpSQLiteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{totpPostgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{totpMySQLInitial()} },
	})
}

func totpSQLiteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260318000000_totp_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS totp (
					id VARCHAR(36) PRIMARY KEY,
					user_id VARCHAR(36) NOT NULL,
					secret TEXT NOT NULL,
					backup_codes TEXT NOT NULL,
					enabled BOOLEAN NOT NULL DEFAULT FALSE,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_totp_user_id ON totp(user_id);`,
				`CREATE TABLE IF NOT EXISTS trusted_devices (
					id VARCHAR(36) PRIMARY KEY,
					user_id VARCHAR(36) NOT NULL,
					token VARCHAR(64) NOT NULL,
					user_agent TEXT NOT NULL DEFAULT '',
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
				`DROP TABLE IF EXISTS trusted_devices;`,
				`DROP TABLE IF EXISTS totp;`,
			)
		},
	}
}

func totpPostgresInitial() migrations.Migration {
	return migrations.Migration{
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
	}
}

func totpMySQLInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260318000000_totp_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS totp (
					id BINARY(16) NOT NULL PRIMARY KEY,
					user_id BINARY(16) NOT NULL,
					secret TEXT NOT NULL,
					backup_codes TEXT NOT NULL,
					enabled BOOLEAN NOT NULL DEFAULT FALSE,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					UNIQUE INDEX idx_totp_user_id (user_id)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				`CREATE TABLE IF NOT EXISTS trusted_devices (
					id BINARY(16) NOT NULL PRIMARY KEY,
					user_id BINARY(16) NOT NULL,
					token VARCHAR(255) NOT NULL,
					user_agent TEXT NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					INDEX idx_trusted_devices_user_id (user_id),
					INDEX idx_trusted_devices_token (token),
					INDEX idx_trusted_devices_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS trusted_devices;`,
				`DROP TABLE IF EXISTS totp;`,
			)
		},
	}
}
