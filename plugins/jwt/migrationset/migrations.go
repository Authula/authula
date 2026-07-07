package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func Migrations() []migrations.Migration {
	return []migrations.Migration{jwtPostgresInitial()}
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
