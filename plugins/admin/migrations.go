package admin

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
)

func adminMigrationsForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{adminSQLiteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{adminPostgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{adminMySQLInitial()} },
	})
}

func adminSQLiteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260222000000_admin_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`PRAGMA foreign_keys = ON;`,
				`CREATE TABLE IF NOT EXISTS admin_impersonations (
          id TEXT PRIMARY KEY,
          actor_user_id TEXT NOT NULL,
          target_user_id TEXT NOT NULL,
          actor_session_id TEXT,
          impersonation_session_id TEXT,
          reason TEXT NOT NULL,
          started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          expires_at TIMESTAMP NOT NULL,
          ended_at TIMESTAMP,
          ended_by_user_id TEXT,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE CASCADE,
          FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE CASCADE,
          FOREIGN KEY (ended_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          FOREIGN KEY (actor_session_id) REFERENCES sessions(id) ON DELETE SET NULL,
          FOREIGN KEY (impersonation_session_id) REFERENCES sessions(id) ON DELETE SET NULL,
          CHECK (actor_user_id != target_user_id)
        );`,
				`CREATE INDEX IF NOT EXISTS idx_admin_impersonations_actor_user_id ON admin_impersonations(actor_user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_impersonations_target_user_id ON admin_impersonations(target_user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_impersonations_expires_at_ended_at ON admin_impersonations(expires_at, ended_at);`,
				`CREATE TABLE IF NOT EXISTS admin_user_states (
          user_id TEXT PRIMARY KEY,
          banned BOOLEAN NOT NULL DEFAULT 0,
          banned_at TIMESTAMP,
          banned_until TIMESTAMP,
          banned_reason TEXT,
          banned_by_user_id TEXT,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
          FOREIGN KEY (banned_by_user_id) REFERENCES users(id) ON DELETE SET NULL
        );`,
				`CREATE INDEX IF NOT EXISTS idx_admin_user_states_banned_banned_until ON admin_user_states(banned, banned_until);`,
				`CREATE TABLE IF NOT EXISTS admin_session_states (
          session_id TEXT PRIMARY KEY,
          revoked_at TIMESTAMP,
          revoked_reason TEXT,
          revoked_by_user_id TEXT,
          impersonator_user_id TEXT,
          impersonation_reason TEXT,
          impersonation_expires_at TIMESTAMP,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
          FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          FOREIGN KEY (impersonator_user_id) REFERENCES users(id) ON DELETE SET NULL
        );`,
				`CREATE INDEX IF NOT EXISTS idx_admin_session_states_revoked_at_impersonation_expires_at ON admin_session_states(revoked_at, impersonation_expires_at);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_session_states_impersonator_user_id ON admin_session_states(impersonator_user_id);`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS admin_session_states;`,
				`DROP TABLE IF EXISTS admin_user_states;`,
				`DROP TABLE IF EXISTS admin_impersonations;`,
			)
		},
	}
}

func adminPostgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260222000000_admin_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE OR REPLACE FUNCTION admin_set_updated_at_fn() RETURNS TRIGGER AS $$
          BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
          END;
        $$ LANGUAGE plpgsql;`,
				`CREATE TABLE IF NOT EXISTS admin_impersonations (
          id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
          actor_user_id UUID NOT NULL,
          target_user_id UUID NOT NULL,
          actor_session_id UUID,
          impersonation_session_id UUID,
          reason TEXT NOT NULL,
          started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
          ended_at TIMESTAMP WITH TIME ZONE,
          ended_by_user_id UUID,
          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          CONSTRAINT fk_admin_impersonations_actor_user FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_impersonations_target_user FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_impersonations_ended_by_user FOREIGN KEY (ended_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          CONSTRAINT fk_admin_impersonations_actor_session FOREIGN KEY (actor_session_id) REFERENCES sessions(id) ON DELETE SET NULL,
          CONSTRAINT fk_admin_impersonations_impersonation_session FOREIGN KEY (impersonation_session_id) REFERENCES sessions(id) ON DELETE SET NULL,
          CONSTRAINT chk_admin_impersonation_actor_target CHECK (actor_user_id <> target_user_id)
        );`,
				`DROP TRIGGER IF EXISTS update_admin_impersonations_updated_at_trigger ON admin_impersonations;`,
				`CREATE TRIGGER update_admin_impersonations_updated_at_trigger
        BEFORE UPDATE ON admin_impersonations
        FOR EACH ROW
        EXECUTE FUNCTION admin_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_admin_impersonations_actor_user_id ON admin_impersonations(actor_user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_impersonations_target_user_id ON admin_impersonations(target_user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_impersonations_expires_at_ended_at ON admin_impersonations(expires_at, ended_at);`,
				`CREATE TABLE IF NOT EXISTS admin_user_states (
          user_id UUID PRIMARY KEY,
          banned BOOLEAN NOT NULL DEFAULT FALSE,
          banned_at TIMESTAMP WITH TIME ZONE,
          banned_until TIMESTAMP WITH TIME ZONE,
          banned_reason TEXT,
          banned_by_user_id UUID,
          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          CONSTRAINT fk_admin_user_states_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_user_states_banned_by FOREIGN KEY (banned_by_user_id) REFERENCES users(id) ON DELETE SET NULL
        );`,
				`DROP TRIGGER IF EXISTS update_admin_user_states_updated_at_trigger ON admin_user_states;`,
				`CREATE TRIGGER update_admin_user_states_updated_at_trigger
        BEFORE UPDATE ON admin_user_states
        FOR EACH ROW
        EXECUTE FUNCTION admin_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_admin_user_states_banned_banned_until ON admin_user_states(banned, banned_until);`,
				`CREATE TABLE IF NOT EXISTS admin_session_states (
          session_id UUID PRIMARY KEY,
          revoked_at TIMESTAMP WITH TIME ZONE,
          revoked_reason TEXT,
          revoked_by_user_id UUID,
          impersonator_user_id UUID,
          impersonation_reason TEXT,
          impersonation_expires_at TIMESTAMP WITH TIME ZONE,
          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          CONSTRAINT fk_admin_session_states_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_session_states_revoked_by FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          CONSTRAINT fk_admin_session_states_impersonator FOREIGN KEY (impersonator_user_id) REFERENCES users(id) ON DELETE SET NULL
        );`,
				`DROP TRIGGER IF EXISTS update_admin_session_states_updated_at_trigger ON admin_session_states;`,
				`CREATE TRIGGER update_admin_session_states_updated_at_trigger
        BEFORE UPDATE ON admin_session_states
        FOR EACH ROW
        EXECUTE FUNCTION admin_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_admin_session_states_revoked_at_impersonation_expires_at ON admin_session_states(revoked_at, impersonation_expires_at);`,
				`CREATE INDEX IF NOT EXISTS idx_admin_session_states_impersonator_user_id ON admin_session_states(impersonator_user_id);`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TRIGGER IF EXISTS update_admin_session_states_updated_at_trigger ON admin_session_states;`,
				`DROP TABLE IF EXISTS admin_session_states;`,
				`DROP TRIGGER IF EXISTS update_admin_user_states_updated_at_trigger ON admin_user_states;`,
				`DROP TABLE IF EXISTS admin_user_states;`,
				`DROP TRIGGER IF EXISTS update_admin_impersonations_updated_at_trigger ON admin_impersonations;`,
				`DROP TABLE IF EXISTS admin_impersonations;`,
				`DROP FUNCTION IF EXISTS admin_set_updated_at_fn();`,
			)
		},
	}
}

func adminMySQLInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260222000000_admin_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS admin_impersonations (
          id BINARY(16) NOT NULL PRIMARY KEY,
          actor_user_id BINARY(16) NOT NULL,
          target_user_id BINARY(16) NOT NULL,
          actor_session_id BINARY(16) NULL,
          impersonation_session_id BINARY(16) NULL,
          reason TEXT NOT NULL,
          started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          expires_at TIMESTAMP NOT NULL,
          ended_at TIMESTAMP NULL,
          ended_by_user_id BINARY(16) NULL,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
          CONSTRAINT fk_admin_impersonations_actor_user FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_impersonations_target_user FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_impersonations_actor_session FOREIGN KEY (actor_session_id) REFERENCES sessions(id) ON DELETE SET NULL,
          CONSTRAINT fk_admin_impersonations_impersonation_session FOREIGN KEY (impersonation_session_id) REFERENCES sessions(id) ON DELETE SET NULL,
          CONSTRAINT fk_admin_impersonations_ended_by_user FOREIGN KEY (ended_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          CONSTRAINT chk_admin_impersonation_actor_target CHECK (actor_user_id <> target_user_id),
          INDEX idx_admin_impersonations_actor_user_id (actor_user_id),
          INDEX idx_admin_impersonations_target_user_id (target_user_id),
          INDEX idx_admin_impersonations_expires_at_ended_at (expires_at, ended_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				`CREATE TABLE IF NOT EXISTS admin_user_states (
          user_id BINARY(16) NOT NULL PRIMARY KEY,
          banned TINYINT(1) NOT NULL DEFAULT 0,
          banned_at TIMESTAMP NULL,
          banned_until TIMESTAMP NULL,
          banned_reason TEXT NULL,
          banned_by_user_id BINARY(16) NULL,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
          CONSTRAINT fk_admin_user_states_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_user_states_banned_by FOREIGN KEY (banned_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          INDEX idx_admin_user_states_banned_banned_until (banned, banned_until)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				`CREATE TABLE IF NOT EXISTS admin_session_states (
          session_id BINARY(16) NOT NULL PRIMARY KEY,
          revoked_at TIMESTAMP NULL,
          revoked_reason TEXT NULL,
          revoked_by_user_id BINARY(16) NULL,
          impersonator_user_id BINARY(16) NULL,
          impersonation_reason TEXT NULL,
          impersonation_expires_at TIMESTAMP NULL,
          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
          CONSTRAINT fk_admin_session_states_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
          CONSTRAINT fk_admin_session_states_revoked_by FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
          CONSTRAINT fk_admin_session_states_impersonator FOREIGN KEY (impersonator_user_id) REFERENCES users(id) ON DELETE SET NULL,
          INDEX idx_admin_session_states_revoked_at_impersonation_expires_at (revoked_at, impersonation_expires_at),
          INDEX idx_admin_session_states_impersonator_user_id (impersonator_user_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS admin_user_states;`,
				`DROP TABLE IF EXISTS admin_session_states;`,
				`DROP TABLE IF EXISTS admin_impersonations;`,
			)
		},
	}
}
