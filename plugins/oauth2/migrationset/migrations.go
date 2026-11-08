package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

// ForProvider returns the OAuth2 authorization server schema migrations for
// the requested database provider.
func ForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{sqliteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{postgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{mysqlInitial()} },
	})
}

func sqliteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20261105000000_oauth2_authorization_server_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`PRAGMA foreign_keys = ON;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_clients (
					id TEXT PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL UNIQUE,
					client_secret_hash VARCHAR(255) NOT NULL,
					client_secret_salt VARCHAR(255) NOT NULL,
					client_secret_prefix VARCHAR(32) NOT NULL,
					name VARCHAR(255) NOT NULL,
					redirect_uris TEXT NOT NULL,
					allowed_grants TEXT NOT NULL,
					allowed_scopes TEXT NOT NULL,
					owner_type VARCHAR(32) NOT NULL,
					owner_id TEXT NOT NULL,
					creator_id TEXT NOT NULL,
					organization_id TEXT,
					is_active BOOLEAN NOT NULL DEFAULT 1,
					deleted_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_clients_owner_type CHECK (owner_type IN ('user', 'organization'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_clients_owner ON oauth_clients(owner_type, owner_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_clients_organization_id ON oauth_clients(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_clients_deleted_at ON oauth_clients(deleted_at);`,
				`DROP TRIGGER IF EXISTS update_oauth_clients_updated_at_trigger;`,
				`CREATE TRIGGER update_oauth_clients_updated_at_trigger
				AFTER UPDATE ON oauth_clients
				FOR EACH ROW
				WHEN NEW.updated_at <= OLD.updated_at
				BEGIN
					UPDATE oauth_clients SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
				END;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS authorization_codes (
					id TEXT PRIMARY KEY,
					code_hash VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id TEXT NOT NULL,
					session_id TEXT NOT NULL,
					consent_id TEXT NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					nonce TEXT,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					auth_time TIMESTAMP NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_authorization_codes_challenge_method CHECK (code_challenge_method = 'S256')
				);`,
				`CREATE INDEX IF NOT EXISTS idx_authorization_codes_client_id ON authorization_codes(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_authorization_codes_user_id ON authorization_codes(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_authorization_codes_expires_at ON authorization_codes(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consent_requests (
					id TEXT PRIMARY KEY,
					request_hash VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id TEXT NOT NULL,
					session_id TEXT NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					state TEXT,
					nonce TEXT,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(32) NOT NULL DEFAULT '',
					max_age INTEGER,
					auth_time TIMESTAMP NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_consent_requests_challenge_method CHECK (code_challenge_method = 'S256')
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consent_requests_user_id ON oauth_consent_requests(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consent_requests_expires_at ON oauth_consent_requests(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consents (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					scope TEXT NOT NULL,
					source VARCHAR(32) NOT NULL,
					revoked_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_consents_source CHECK (source IN ('explicit', 'trusted', 'prompt_none'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consents_user_client ON oauth_consents(user_id, client_id);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_consents_active_user_client ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consents_revoked_at ON oauth_consents(revoked_at);`,
				`DROP TRIGGER IF EXISTS update_oauth_consents_updated_at_trigger;`,
				`CREATE TRIGGER update_oauth_consents_updated_at_trigger
				AFTER UPDATE ON oauth_consents
				FOR EACH ROW
				WHEN NEW.updated_at <= OLD.updated_at
				BEGIN
					UPDATE oauth_consents SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
				END;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_login_transactions (
					id TEXT PRIMARY KEY,
					transaction_hash VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					state TEXT,
					nonce TEXT,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(32) NOT NULL DEFAULT '',
					max_age INTEGER,
					auth_time TIMESTAMP NOT NULL,
					completed_at TIMESTAMP,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_login_transactions_challenge_method CHECK (code_challenge_method = 'S256')
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_login_transactions_expires_at ON oauth_login_transactions(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_flow_states (
					id TEXT PRIMARY KEY,
					state_hash VARCHAR(64) UNIQUE NOT NULL,
					provider VARCHAR(255) NOT NULL,
					purpose VARCHAR(32) NOT NULL,
					user_id TEXT,
					redirect_to TEXT,
					code_verifier_encrypted TEXT,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_flow_states_provider ON oauth_flow_states(provider);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_flow_states_expires_at ON oauth_flow_states(expires_at);`,
				// -----------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TRIGGER IF EXISTS update_oauth_consents_updated_at_trigger;`,
				`DROP TRIGGER IF EXISTS update_oauth_clients_updated_at_trigger;`,
				`DROP TABLE IF EXISTS oauth_flow_states;`,
				`DROP TABLE IF EXISTS oauth_login_transactions;`,
				`DROP TABLE IF EXISTS oauth_consents;`,
				`DROP TABLE IF EXISTS oauth_consent_requests;`,
				`DROP TABLE IF EXISTS authorization_codes;`,
				`DROP TABLE IF EXISTS oauth_clients;`,
			)
		},
	}
}

func postgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20261105000000_oauth2_authorization_server_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE OR REPLACE FUNCTION oauth2_set_updated_at_fn()
				RETURNS TRIGGER AS $$
				BEGIN
					NEW.updated_at = NOW();
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_clients (
					id UUID PRIMARY KEY,
					client_id VARCHAR(255) UNIQUE NOT NULL,
					client_secret_hash VARCHAR(255) NOT NULL,
					client_secret_salt VARCHAR(255) NOT NULL,
					client_secret_prefix VARCHAR(32) NOT NULL,
					name VARCHAR(255) NOT NULL,
					redirect_uris JSONB NOT NULL DEFAULT '[]'::jsonb,
					allowed_grants JSONB NOT NULL DEFAULT '[]'::jsonb,
					allowed_scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
					owner_type VARCHAR(32) NOT NULL,
					owner_id UUID NOT NULL,
					creator_id UUID NOT NULL,
					organization_id UUID,
					is_active BOOLEAN NOT NULL DEFAULT TRUE,
					deleted_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_oauth_clients_owner_type CHECK (owner_type IN ('user', 'organization'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_clients_owner ON oauth_clients(owner_type, owner_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_clients_organization_id ON oauth_clients(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_clients_deleted_at ON oauth_clients(deleted_at);`,
				`DROP TRIGGER IF EXISTS update_oauth_clients_updated_at_trigger ON oauth_clients;`,
				`CREATE TRIGGER update_oauth_clients_updated_at_trigger
				BEFORE UPDATE ON oauth_clients
				FOR EACH ROW
				EXECUTE FUNCTION oauth2_set_updated_at_fn();`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS authorization_codes (
					id UUID PRIMARY KEY,
					code_hash VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id UUID NOT NULL,
					session_id UUID NOT NULL,
					consent_id UUID NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					nonce TEXT,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					auth_time TIMESTAMP WITH TIME ZONE NOT NULL,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					consumed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_authorization_codes_challenge_method CHECK (code_challenge_method = 'S256')
				);`,
				`CREATE INDEX IF NOT EXISTS idx_authorization_codes_client_id ON authorization_codes(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_authorization_codes_user_id ON authorization_codes(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_authorization_codes_expires_at ON authorization_codes(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consent_requests (
					id UUID PRIMARY KEY,
					request_hash VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id UUID NOT NULL,
					session_id UUID NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					state TEXT,
					nonce TEXT,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(32) NOT NULL DEFAULT '',
					max_age INTEGER,
					auth_time TIMESTAMP WITH TIME ZONE NOT NULL,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					consumed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_oauth_consent_requests_challenge_method CHECK (code_challenge_method = 'S256')
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consent_requests_user_id ON oauth_consent_requests(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consent_requests_expires_at ON oauth_consent_requests(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consents (
					id UUID PRIMARY KEY,
					user_id UUID NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					scope TEXT NOT NULL,
					source VARCHAR(32) NOT NULL,
					revoked_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_oauth_consents_source CHECK (source IN ('explicit', 'trusted', 'prompt_none'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consents_user_client ON oauth_consents(user_id, client_id);`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_consents_active_user_client ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consents_revoked_at ON oauth_consents(revoked_at);`,
				`DROP TRIGGER IF EXISTS update_oauth_consents_updated_at_trigger ON oauth_consents;`,
				`CREATE TRIGGER update_oauth_consents_updated_at_trigger
				BEFORE UPDATE ON oauth_consents
				FOR EACH ROW
				EXECUTE FUNCTION oauth2_set_updated_at_fn();`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_login_transactions (
					id UUID PRIMARY KEY,
					transaction_hash VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					state TEXT,
					nonce TEXT,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(32) NOT NULL DEFAULT '',
					max_age INTEGER,
					auth_time TIMESTAMP WITH TIME ZONE NOT NULL,
					completed_at TIMESTAMP WITH TIME ZONE,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					consumed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_oauth_login_transactions_challenge_method CHECK (code_challenge_method = 'S256')
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_login_transactions_expires_at ON oauth_login_transactions(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_flow_states (
					id UUID PRIMARY KEY,
					state_hash VARCHAR(64) UNIQUE NOT NULL,
					provider VARCHAR(255) NOT NULL,
					purpose VARCHAR(32) NOT NULL,
					user_id UUID,
					redirect_to TEXT,
					code_verifier_encrypted TEXT,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					consumed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_flow_states_provider ON oauth_flow_states(provider);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_flow_states_expires_at ON oauth_flow_states(expires_at);`,
				// -----------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TRIGGER IF EXISTS update_oauth_consents_updated_at_trigger ON oauth_consents;`,
				`DROP TRIGGER IF EXISTS update_oauth_clients_updated_at_trigger ON oauth_clients;`,
				`DROP TABLE IF EXISTS oauth_flow_states;`,
				`DROP TABLE IF EXISTS oauth_login_transactions;`,
				`DROP TABLE IF EXISTS oauth_consents;`,
				`DROP TABLE IF EXISTS oauth_consent_requests;`,
				`DROP TABLE IF EXISTS authorization_codes;`,
				`DROP TABLE IF EXISTS oauth_clients;`,
				`DROP FUNCTION IF EXISTS oauth2_set_updated_at_fn();`,
			)
		},
	}
}

func mysqlInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20261105000000_oauth2_authorization_server_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_clients (
					id BINARY(16) NOT NULL PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL,
					client_secret_hash VARCHAR(255) NOT NULL,
					client_secret_salt VARCHAR(255) NOT NULL,
					client_secret_prefix VARCHAR(32) NOT NULL,
					name VARCHAR(255) NOT NULL,
					redirect_uris JSON NOT NULL,
					allowed_grants JSON NOT NULL,
					allowed_scopes JSON NOT NULL,
					owner_type VARCHAR(32) NOT NULL,
					owner_id BINARY(16) NOT NULL,
					creator_id BINARY(16) NOT NULL,
					organization_id BINARY(16) NULL,
					is_active TINYINT(1) NOT NULL DEFAULT 1,
					deleted_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_clients_owner_type CHECK (owner_type IN ('user', 'organization')),
					UNIQUE KEY idx_oauth_clients_client_id (client_id),
					INDEX idx_oauth_clients_owner (owner_type, owner_id),
					INDEX idx_oauth_clients_organization_id (organization_id),
					INDEX idx_oauth_clients_deleted_at (deleted_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS authorization_codes (
					id BINARY(16) NOT NULL PRIMARY KEY,
					code_hash VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id BINARY(16) NOT NULL,
					session_id BINARY(16) NOT NULL,
					consent_id BINARY(16) NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					nonce TEXT NULL,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					auth_time TIMESTAMP NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_authorization_codes_challenge_method CHECK (code_challenge_method = 'S256'),
					UNIQUE KEY idx_authorization_codes_code_hash (code_hash),
					INDEX idx_authorization_codes_client_id (client_id),
					INDEX idx_authorization_codes_user_id (user_id),
					INDEX idx_authorization_codes_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consent_requests (
					id BINARY(16) NOT NULL PRIMARY KEY,
					request_hash VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id BINARY(16) NOT NULL,
					session_id BINARY(16) NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					state TEXT NULL,
					nonce TEXT NULL,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(32) NOT NULL DEFAULT '',
					max_age INTEGER NULL,
					auth_time TIMESTAMP NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_consent_requests_challenge_method CHECK (code_challenge_method = 'S256'),
					UNIQUE KEY idx_oauth_consent_requests_request_hash (request_hash),
					INDEX idx_oauth_consent_requests_user_id (user_id),
					INDEX idx_oauth_consent_requests_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consents (
					id BINARY(16) NOT NULL PRIMARY KEY,
					user_id BINARY(16) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					scope TEXT NOT NULL,
					source VARCHAR(32) NOT NULL,
					revoked_at TIMESTAMP NULL DEFAULT NULL,
					active_client_id VARCHAR(255) GENERATED ALWAYS AS (IF(revoked_at IS NULL, client_id, NULL)) STORED,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_consents_source CHECK (source IN ('explicit', 'trusted', 'prompt_none')),
					INDEX idx_oauth_consents_user_client (user_id, client_id),
					UNIQUE KEY idx_oauth_consents_active_user_client (user_id, active_client_id),
					INDEX idx_oauth_consents_revoked_at (revoked_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_login_transactions (
					id BINARY(16) NOT NULL PRIMARY KEY,
					transaction_hash VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					redirect_uri TEXT NOT NULL,
					scope TEXT NOT NULL,
					state TEXT NULL,
					nonce TEXT NULL,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(32) NOT NULL DEFAULT '',
					max_age INTEGER NULL,
					auth_time TIMESTAMP NOT NULL,
					completed_at TIMESTAMP NULL DEFAULT NULL,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_login_transactions_challenge_method CHECK (code_challenge_method = 'S256'),
					UNIQUE KEY idx_oauth_login_transactions_transaction_hash (transaction_hash),
					INDEX idx_oauth_login_transactions_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_flow_states (
					id BINARY(16) NOT NULL PRIMARY KEY,
					state_hash VARCHAR(64) NOT NULL,
					provider VARCHAR(255) NOT NULL,
					purpose VARCHAR(32) NOT NULL,
					user_id BINARY(16) NULL,
					redirect_to TEXT NULL,
					code_verifier_encrypted TEXT,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					UNIQUE KEY idx_oauth_flow_states_state_hash (state_hash),
					INDEX idx_oauth_flow_states_provider (provider),
					INDEX idx_oauth_flow_states_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS oauth_flow_states;`,
				`DROP TABLE IF EXISTS oauth_login_transactions;`,
				`DROP TABLE IF EXISTS oauth_consents;`,
				`DROP TABLE IF EXISTS oauth_consent_requests;`,
				`DROP TABLE IF EXISTS authorization_codes;`,
				`DROP TABLE IF EXISTS oauth_clients;`,
			)
		},
	}
}
