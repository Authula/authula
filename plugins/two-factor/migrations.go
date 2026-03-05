package twofactor

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
)

func twoFactorMigrationsForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{twoFactorSQLiteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{twoFactorPostgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{twoFactorMySQLInitial()} },
	})
}

func twoFactorSQLiteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260304000000_two_factor_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`ALTER TABLE users ADD COLUMN two_factor_enabled BOOLEAN DEFAULT FALSE;`,
				`CREATE TABLE IF NOT EXISTS two_factor (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  secret TEXT NOT NULL,
  backup_codes TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_two_factor_user_id ON two_factor(user_id);`,
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
				`DROP TABLE IF EXISTS two_factor;`,
				`ALTER TABLE users DROP COLUMN two_factor_enabled;`,
			)
		},
	}
}

func twoFactorPostgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260304000000_two_factor_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN DEFAULT FALSE;`,
				`CREATE OR REPLACE FUNCTION two_factor_update_updated_at_func()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;`,
				`CREATE TABLE IF NOT EXISTS two_factor (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  secret TEXT NOT NULL,
  backup_codes TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_two_factor_user_id ON two_factor(user_id);`,
				`DROP TRIGGER IF EXISTS two_factor_update_updated_at_trigger ON two_factor;`,
				`CREATE TRIGGER two_factor_update_updated_at_trigger
  BEFORE UPDATE ON two_factor
  FOR EACH ROW
  EXECUTE FUNCTION two_factor_update_updated_at_func();`,
				`CREATE TABLE IF NOT EXISTS trusted_devices (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
				`DROP TRIGGER IF EXISTS two_factor_update_updated_at_trigger ON two_factor;`,
				`DROP FUNCTION IF EXISTS two_factor_update_updated_at_func();`,
				`DROP TABLE IF EXISTS trusted_devices;`,
				`DROP TABLE IF EXISTS two_factor;`,
				`ALTER TABLE users DROP COLUMN IF EXISTS two_factor_enabled;`,
			)
		},
	}
}

func twoFactorMySQLInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260304000000_two_factor_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`ALTER TABLE users ADD COLUMN two_factor_enabled BOOLEAN DEFAULT FALSE;`,
				`CREATE TABLE IF NOT EXISTS two_factor (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  secret TEXT NOT NULL,
  backup_codes TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE INDEX idx_two_factor_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				`CREATE TABLE IF NOT EXISTS trusted_devices (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  token VARCHAR(64) NOT NULL,
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
				`DROP TABLE IF EXISTS two_factor;`,
				`ALTER TABLE users DROP COLUMN two_factor_enabled;`,
			)
		},
	}
}
