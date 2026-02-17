package jwt

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
)

func jwtMigrationsForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{jwtSQLiteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{jwtPostgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{jwtMySQLInitial()} },
	})
}

func jwtSQLiteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260131000000_jwt_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS jwks (
  id TEXT PRIMARY KEY,
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NULL
);`,
				`CREATE INDEX IF NOT EXISTS idx_jwks_expires_at ON jwks(expires_at);`,
				`CREATE TABLE IF NOT EXISTS refresh_tokens (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  is_revoked INTEGER DEFAULT 0,
  revoked_at TIMESTAMP NULL,
  last_reuse_attempt TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_id ON refresh_tokens(session_id);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked_only ON refresh_tokens(is_revoked) WHERE is_revoked = 1;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS refresh_tokens;`,
				`DROP TABLE IF EXISTS jwks;`,
			)
		},
	}
}

func jwtPostgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260131000000_jwt_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS jwks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP WITH TIME ZONE NULL
);`,
				`CREATE INDEX IF NOT EXISTS idx_jwks_expires_at ON jwks(expires_at);`,
				`CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL,
  token_hash VARCHAR(64) UNIQUE NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  is_revoked BOOLEAN DEFAULT FALSE,
  revoked_at TIMESTAMP WITH TIME ZONE NULL,
  last_reuse_attempt TIMESTAMP WITH TIME ZONE NULL DEFAULT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT fk_refresh_tokens_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_id ON refresh_tokens(session_id);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);`,
				`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked_only ON refresh_tokens(is_revoked) WHERE is_revoked = TRUE;`,
				`CREATE OR REPLACE FUNCTION cleanup_expired_refresh_tokens()
RETURNS VOID AS $$
BEGIN
  DELETE FROM refresh_tokens WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP FUNCTION IF EXISTS cleanup_expired_refresh_tokens();`,
				`DROP TABLE IF EXISTS refresh_tokens;`,
				`DROP TABLE IF EXISTS jwks;`,
			)
		},
	}
}

func jwtMySQLInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260131000000_jwt_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS jwks (
  id BINARY(16) NOT NULL PRIMARY KEY,
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NULL
);`,
				`CREATE INDEX idx_jwks_expires_at ON jwks(expires_at);`,
				`CREATE TABLE IF NOT EXISTS refresh_tokens (
  id BINARY(16) NOT NULL PRIMARY KEY,
  session_id BINARY(16) NOT NULL,
  token_hash VARCHAR(64) UNIQUE NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  is_revoked BOOLEAN DEFAULT FALSE,
  revoked_at TIMESTAMP NULL,
  last_reuse_attempt TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_refresh_tokens_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);`,
				`CREATE INDEX idx_refresh_tokens_session_id ON refresh_tokens(session_id);`,
				`CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);`,
				`CREATE INDEX idx_refresh_tokens_active_session ON refresh_tokens(session_id, is_revoked);`,
				`CREATE INDEX idx_refresh_tokens_last_reuse_attempt ON refresh_tokens(last_reuse_attempt);`,
				`DROP PROCEDURE IF EXISTS cleanup_expired_refresh_tokens;`,
				`CREATE PROCEDURE cleanup_expired_refresh_tokens()
BEGIN
  DELETE FROM refresh_tokens WHERE expires_at < NOW();
END;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP PROCEDURE IF EXISTS cleanup_expired_refresh_tokens;`,
				`DROP TABLE IF EXISTS refresh_tokens;`,
				`DROP TABLE IF EXISTS jwks;`,
			)
		},
	}
}
