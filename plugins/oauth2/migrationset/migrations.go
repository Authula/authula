package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

const oauth2InitialVersion = "20261105000000_oauth2_authorization_server_initial"

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
		Version: oauth2InitialVersion,
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`PRAGMA foreign_keys = ON;`,
				`CREATE TABLE IF NOT EXISTS oauth_clients (
					id TEXT PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL UNIQUE,
					client_secret_hash VARCHAR(255) NOT NULL,
					client_secret_salt VARCHAR(255) NOT NULL,
					client_secret_prefix VARCHAR(32) NOT NULL,
					name VARCHAR(255) NOT NULL,
					redirect_uris TEXT NOT NULL,
					post_logout_redirect_uris TEXT NOT NULL DEFAULT '[]',
					allowed_grants TEXT NOT NULL,
					allowed_scopes TEXT NOT NULL,
					allowed_audiences TEXT NOT NULL DEFAULT '[]',
					owner_type VARCHAR(32) NOT NULL,
					owner_id TEXT NOT NULL,
					creator_id TEXT NOT NULL,
					organization_id TEXT,
					application_type VARCHAR(16) NOT NULL DEFAULT 'web',
					auth_method VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic',
					trusted_first_party BOOLEAN NOT NULL DEFAULT 0,
					require_consent BOOLEAN NOT NULL DEFAULT 1,
					access_token_type VARCHAR(16) NOT NULL DEFAULT 'jwt',
					access_token_lifetime INTEGER NOT NULL DEFAULT 900,
					id_token_lifetime INTEGER NOT NULL DEFAULT 300,
					login_url TEXT,
					jwks_uri TEXT,
					token_endpoint_auth_signing_alg VARCHAR(16),
					is_active BOOLEAN NOT NULL DEFAULT 1,
					deleted_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_clients_owner_type CHECK (owner_type IN ('user', 'organization')),
					CONSTRAINT chk_oauth_clients_application_type CHECK (application_type IN ('web', 'native')),
					CONSTRAINT chk_oauth_clients_auth_method CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'none')),
					CONSTRAINT chk_oauth_clients_access_token_type CHECK (access_token_type IN ('jwt', 'opaque'))
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
				`CREATE TABLE IF NOT EXISTS oauth_auth_requests (
					id TEXT PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL,
					user_id TEXT,
					session_id TEXT,
					redirect_uri TEXT NOT NULL,
					scopes TEXT NOT NULL,
					audience TEXT,
					state TEXT,
					nonce TEXT,
					response_type VARCHAR(64) NOT NULL DEFAULT 'code',
					response_mode VARCHAR(32),
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(128) NOT NULL DEFAULT '',
					max_age INTEGER,
					ui_locales TEXT,
					login_hint TEXT,
					acr_values TEXT,
					auth_time TIMESTAMP,
					amr TEXT,
					acr VARCHAR(255),
					binding_hash VARCHAR(64) NOT NULL UNIQUE,
					consent_status VARCHAR(16) NOT NULL DEFAULT 'pending',
					code_hash VARCHAR(64) UNIQUE,
					completed_at TIMESTAMP,
					code_issued_at TIMESTAMP,
					code_consumed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					expires_at TIMESTAMP NOT NULL,
					CONSTRAINT chk_oauth_auth_requests_challenge_method CHECK (code_challenge_method = 'S256'),
					CONSTRAINT chk_oauth_auth_requests_consent_status CHECK (consent_status IN ('pending', 'granted', 'denied'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_client_id ON oauth_auth_requests(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_user_id ON oauth_auth_requests(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_session_id ON oauth_auth_requests(session_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_expires_at ON oauth_auth_requests(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consent_requests (
					id TEXT PRIMARY KEY,
					token_hash VARCHAR(64) NOT NULL UNIQUE,
					auth_request_id TEXT NOT NULL,
					user_id TEXT NOT NULL,
					session_id TEXT NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					requested_scopes TEXT NOT NULL,
					decision VARCHAR(16),
					granted_scopes TEXT,
					completion_code_hash VARCHAR(64) UNIQUE,
					origin TEXT,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_consent_requests_decision CHECK (decision IS NULL OR decision IN ('allow', 'deny'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consent_requests_auth_request_id ON oauth_consent_requests(auth_request_id);`,
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
				`CREATE TABLE IF NOT EXISTS oauth_access_grants (
					id TEXT PRIMARY KEY,
					jti VARCHAR(64) NOT NULL UNIQUE,
					client_id VARCHAR(255) NOT NULL,
					user_id TEXT NOT NULL,
					session_id TEXT NOT NULL,
					family_id VARCHAR(64) NOT NULL,
					scopes TEXT NOT NULL,
					audience TEXT,
					auth_time TIMESTAMP,
					amr TEXT,
					expires_at TIMESTAMP NOT NULL,
					revoked_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_family_id ON oauth_access_grants(family_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_client_id ON oauth_access_grants(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_user_id ON oauth_access_grants(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_expires_at ON oauth_access_grants(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
					id TEXT PRIMARY KEY,
					token_hash VARCHAR(64) NOT NULL UNIQUE,
					family_id VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id TEXT NOT NULL,
					session_id TEXT NOT NULL,
					auth_request_id TEXT,
					scopes TEXT NOT NULL,
					audience TEXT,
					auth_time TIMESTAMP,
					amr TEXT,
					rotated_from VARCHAR(64),
					consumed_at TIMESTAMP,
					revoked_at TIMESTAMP,
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_family_id ON oauth_refresh_tokens(family_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_client_id ON oauth_refresh_tokens(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_user_id ON oauth_refresh_tokens(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_session_id ON oauth_refresh_tokens(session_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_expires_at ON oauth_refresh_tokens(expires_at);`,
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
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TRIGGER IF EXISTS update_oauth_consents_updated_at_trigger;`,
				`DROP TRIGGER IF EXISTS update_oauth_clients_updated_at_trigger;`,
				`DROP TABLE IF EXISTS oauth_flow_states;`,
				`DROP TABLE IF EXISTS oauth_refresh_tokens;`,
				`DROP TABLE IF EXISTS oauth_access_grants;`,
				`DROP TABLE IF EXISTS oauth_consents;`,
				`DROP TABLE IF EXISTS oauth_consent_requests;`,
				`DROP TABLE IF EXISTS oauth_auth_requests;`,
				`DROP TABLE IF EXISTS oauth_clients;`,
			)
		},
	}
}

func postgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: oauth2InitialVersion,
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
				`CREATE TABLE IF NOT EXISTS oauth_clients (
					id UUID PRIMARY KEY,
					client_id VARCHAR(255) UNIQUE NOT NULL,
					client_secret_hash VARCHAR(255) NOT NULL,
					client_secret_salt VARCHAR(255) NOT NULL,
					client_secret_prefix VARCHAR(32) NOT NULL,
					name VARCHAR(255) NOT NULL,
					redirect_uris JSONB NOT NULL DEFAULT '[]'::jsonb,
					post_logout_redirect_uris JSONB NOT NULL DEFAULT '[]'::jsonb,
					allowed_grants JSONB NOT NULL DEFAULT '[]'::jsonb,
					allowed_scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
					allowed_audiences JSONB NOT NULL DEFAULT '[]'::jsonb,
					owner_type VARCHAR(32) NOT NULL,
					owner_id UUID NOT NULL,
					creator_id UUID NOT NULL,
					organization_id UUID,
					application_type VARCHAR(16) NOT NULL DEFAULT 'web',
					auth_method VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic',
					trusted_first_party BOOLEAN NOT NULL DEFAULT FALSE,
					require_consent BOOLEAN NOT NULL DEFAULT TRUE,
					access_token_type VARCHAR(16) NOT NULL DEFAULT 'jwt',
					access_token_lifetime INTEGER NOT NULL DEFAULT 900,
					id_token_lifetime INTEGER NOT NULL DEFAULT 300,
					login_url TEXT,
					jwks_uri TEXT,
					token_endpoint_auth_signing_alg VARCHAR(16),
					is_active BOOLEAN NOT NULL DEFAULT TRUE,
					deleted_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_oauth_clients_owner_type CHECK (owner_type IN ('user', 'organization')),
					CONSTRAINT chk_oauth_clients_application_type CHECK (application_type IN ('web', 'native')),
					CONSTRAINT chk_oauth_clients_auth_method CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'none')),
					CONSTRAINT chk_oauth_clients_access_token_type CHECK (access_token_type IN ('jwt', 'opaque'))
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
				`CREATE TABLE IF NOT EXISTS oauth_auth_requests (
					id UUID PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL,
					user_id UUID,
					session_id UUID,
					redirect_uri TEXT NOT NULL,
					scopes TEXT NOT NULL,
					audience TEXT,
					state TEXT,
					nonce TEXT,
					response_type VARCHAR(64) NOT NULL DEFAULT 'code',
					response_mode VARCHAR(32),
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(128) NOT NULL DEFAULT '',
					max_age INTEGER,
					ui_locales TEXT,
					login_hint TEXT,
					acr_values TEXT,
					auth_time TIMESTAMP WITH TIME ZONE,
					amr JSONB,
					acr VARCHAR(255),
					binding_hash VARCHAR(64) UNIQUE NOT NULL,
					consent_status VARCHAR(16) NOT NULL DEFAULT 'pending',
					code_hash VARCHAR(64) UNIQUE,
					completed_at TIMESTAMP WITH TIME ZONE,
					code_issued_at TIMESTAMP WITH TIME ZONE,
					code_consumed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					CONSTRAINT chk_oauth_auth_requests_challenge_method CHECK (code_challenge_method = 'S256'),
					CONSTRAINT chk_oauth_auth_requests_consent_status CHECK (consent_status IN ('pending', 'granted', 'denied'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_client_id ON oauth_auth_requests(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_user_id ON oauth_auth_requests(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_session_id ON oauth_auth_requests(session_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_auth_requests_expires_at ON oauth_auth_requests(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consent_requests (
					id UUID PRIMARY KEY,
					token_hash VARCHAR(64) UNIQUE NOT NULL,
					auth_request_id UUID NOT NULL,
					user_id UUID NOT NULL,
					session_id UUID NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					requested_scopes TEXT NOT NULL,
					decision VARCHAR(16),
					granted_scopes TEXT,
					completion_code_hash VARCHAR(64) UNIQUE,
					origin TEXT,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					consumed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT chk_oauth_consent_requests_decision CHECK (decision IS NULL OR decision IN ('allow', 'deny'))
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_consent_requests_auth_request_id ON oauth_consent_requests(auth_request_id);`,
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
				`CREATE TABLE IF NOT EXISTS oauth_access_grants (
					id UUID PRIMARY KEY,
					jti VARCHAR(64) UNIQUE NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id UUID NOT NULL,
					session_id UUID NOT NULL,
					family_id VARCHAR(64) NOT NULL,
					scopes TEXT NOT NULL,
					audience TEXT,
					auth_time TIMESTAMP WITH TIME ZONE,
					amr JSONB,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					revoked_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_family_id ON oauth_access_grants(family_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_client_id ON oauth_access_grants(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_user_id ON oauth_access_grants(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_access_grants_expires_at ON oauth_access_grants(expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
					id UUID PRIMARY KEY,
					token_hash VARCHAR(64) UNIQUE NOT NULL,
					family_id VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id UUID NOT NULL,
					session_id UUID NOT NULL,
					auth_request_id UUID,
					scopes TEXT NOT NULL,
					audience TEXT,
					auth_time TIMESTAMP WITH TIME ZONE,
					amr JSONB,
					rotated_from VARCHAR(64),
					consumed_at TIMESTAMP WITH TIME ZONE,
					revoked_at TIMESTAMP WITH TIME ZONE,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
				);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_family_id ON oauth_refresh_tokens(family_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_client_id ON oauth_refresh_tokens(client_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_user_id ON oauth_refresh_tokens(user_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_session_id ON oauth_refresh_tokens(session_id);`,
				`CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_expires_at ON oauth_refresh_tokens(expires_at);`,
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
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TRIGGER IF EXISTS update_oauth_consents_updated_at_trigger ON oauth_consents;`,
				`DROP TRIGGER IF EXISTS update_oauth_clients_updated_at_trigger ON oauth_clients;`,
				`DROP TABLE IF EXISTS oauth_flow_states;`,
				`DROP TABLE IF EXISTS oauth_refresh_tokens;`,
				`DROP TABLE IF EXISTS oauth_access_grants;`,
				`DROP TABLE IF EXISTS oauth_consents;`,
				`DROP TABLE IF EXISTS oauth_consent_requests;`,
				`DROP TABLE IF EXISTS oauth_auth_requests;`,
				`DROP TABLE IF EXISTS oauth_clients;`,
				`DROP FUNCTION IF EXISTS oauth2_set_updated_at_fn();`,
			)
		},
	}
}

func mysqlInitial() migrations.Migration {
	return migrations.Migration{
		Version: oauth2InitialVersion,
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE TABLE IF NOT EXISTS oauth_clients (
					id BINARY(16) NOT NULL PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL,
					client_secret_hash VARCHAR(255) NOT NULL,
					client_secret_salt VARCHAR(255) NOT NULL,
					client_secret_prefix VARCHAR(32) NOT NULL,
					name VARCHAR(255) NOT NULL,
					redirect_uris JSON NOT NULL,
					post_logout_redirect_uris JSON NOT NULL,
					allowed_grants JSON NOT NULL,
					allowed_scopes JSON NOT NULL,
					allowed_audiences JSON NOT NULL,
					owner_type VARCHAR(32) NOT NULL,
					owner_id BINARY(16) NOT NULL,
					creator_id BINARY(16) NOT NULL,
					organization_id BINARY(16) NULL,
					application_type VARCHAR(16) NOT NULL DEFAULT 'web',
					auth_method VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic',
					trusted_first_party TINYINT(1) NOT NULL DEFAULT 0,
					require_consent TINYINT(1) NOT NULL DEFAULT 1,
					access_token_type VARCHAR(16) NOT NULL DEFAULT 'jwt',
					access_token_lifetime INTEGER NOT NULL DEFAULT 900,
					id_token_lifetime INTEGER NOT NULL DEFAULT 300,
					login_url TEXT NULL,
					jwks_uri TEXT NULL,
					token_endpoint_auth_signing_alg VARCHAR(16) NULL,
					is_active TINYINT(1) NOT NULL DEFAULT 1,
					deleted_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_clients_owner_type CHECK (owner_type IN ('user', 'organization')),
					CONSTRAINT chk_oauth_clients_application_type CHECK (application_type IN ('web', 'native')),
					CONSTRAINT chk_oauth_clients_auth_method CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'none')),
					CONSTRAINT chk_oauth_clients_access_token_type CHECK (access_token_type IN ('jwt', 'opaque')),
					UNIQUE KEY idx_oauth_clients_client_id (client_id),
					INDEX idx_oauth_clients_owner (owner_type, owner_id),
					INDEX idx_oauth_clients_organization_id (organization_id),
					INDEX idx_oauth_clients_deleted_at (deleted_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_auth_requests (
					id VARCHAR(36) NOT NULL PRIMARY KEY,
					client_id VARCHAR(255) NOT NULL,
					user_id VARCHAR(36) NULL,
					session_id VARCHAR(36) NULL,
					redirect_uri TEXT NOT NULL,
					scopes TEXT NOT NULL,
					audience TEXT NULL,
					state TEXT NULL,
					nonce TEXT NULL,
					response_type VARCHAR(64) NOT NULL DEFAULT 'code',
					response_mode VARCHAR(32) NULL,
					code_challenge VARCHAR(255) NOT NULL,
					code_challenge_method VARCHAR(16) NOT NULL,
					prompt VARCHAR(128) NOT NULL DEFAULT '',
					max_age INTEGER NULL,
					ui_locales TEXT NULL,
					login_hint TEXT NULL,
					acr_values TEXT NULL,
					auth_time TIMESTAMP NULL DEFAULT NULL,
					amr JSON NULL,
					acr VARCHAR(255) NULL,
					binding_hash VARCHAR(64) NOT NULL,
					consent_status VARCHAR(16) NOT NULL DEFAULT 'pending',
					code_hash VARCHAR(64) NULL,
					completed_at TIMESTAMP NULL DEFAULT NULL,
					code_issued_at TIMESTAMP NULL DEFAULT NULL,
					code_consumed_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					expires_at TIMESTAMP NOT NULL,
					CONSTRAINT chk_oauth_auth_requests_challenge_method CHECK (code_challenge_method = 'S256'),
					CONSTRAINT chk_oauth_auth_requests_consent_status CHECK (consent_status IN ('pending', 'granted', 'denied')),
					UNIQUE KEY idx_oauth_auth_requests_binding_hash (binding_hash),
					UNIQUE KEY idx_oauth_auth_requests_code_hash (code_hash),
					INDEX idx_oauth_auth_requests_client_id (client_id),
					INDEX idx_oauth_auth_requests_user_id (user_id),
					INDEX idx_oauth_auth_requests_session_id (session_id),
					INDEX idx_oauth_auth_requests_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_consent_requests (
					id VARCHAR(36) NOT NULL PRIMARY KEY,
					token_hash VARCHAR(64) NOT NULL,
					auth_request_id VARCHAR(36) NOT NULL,
					user_id VARCHAR(36) NOT NULL,
					session_id VARCHAR(36) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					requested_scopes TEXT NOT NULL,
					decision VARCHAR(16) NULL,
					granted_scopes TEXT NULL,
					completion_code_hash VARCHAR(64) NULL,
					origin TEXT NULL,
					expires_at TIMESTAMP NOT NULL,
					consumed_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT chk_oauth_consent_requests_decision CHECK (decision IS NULL OR decision IN ('allow', 'deny')),
					UNIQUE KEY idx_oauth_consent_requests_token_hash (token_hash),
					UNIQUE KEY idx_oauth_consent_requests_completion_code_hash (completion_code_hash),
					INDEX idx_oauth_consent_requests_auth_request_id (auth_request_id),
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
				`CREATE TABLE IF NOT EXISTS oauth_access_grants (
					id VARCHAR(36) NOT NULL PRIMARY KEY,
					jti VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id VARCHAR(36) NOT NULL,
					session_id VARCHAR(36) NOT NULL,
					family_id VARCHAR(64) NOT NULL,
					scopes TEXT NOT NULL,
					audience TEXT NULL,
					auth_time TIMESTAMP NULL DEFAULT NULL,
					amr JSON NULL,
					expires_at TIMESTAMP NOT NULL,
					revoked_at TIMESTAMP NULL DEFAULT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					UNIQUE KEY idx_oauth_access_grants_jti (jti),
					INDEX idx_oauth_access_grants_family_id (family_id),
					INDEX idx_oauth_access_grants_client_id (client_id),
					INDEX idx_oauth_access_grants_user_id (user_id),
					INDEX idx_oauth_access_grants_expires_at (expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
					id VARCHAR(36) NOT NULL PRIMARY KEY,
					token_hash VARCHAR(64) NOT NULL,
					family_id VARCHAR(64) NOT NULL,
					client_id VARCHAR(255) NOT NULL,
					user_id VARCHAR(36) NOT NULL,
					session_id VARCHAR(36) NOT NULL,
					auth_request_id VARCHAR(36) NULL,
					scopes TEXT NOT NULL,
					audience TEXT NULL,
					auth_time TIMESTAMP NULL DEFAULT NULL,
					amr JSON NULL,
					rotated_from VARCHAR(64) NULL,
					consumed_at TIMESTAMP NULL DEFAULT NULL,
					revoked_at TIMESTAMP NULL DEFAULT NULL,
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					UNIQUE KEY idx_oauth_refresh_tokens_token_hash (token_hash),
					INDEX idx_oauth_refresh_tokens_family_id (family_id),
					INDEX idx_oauth_refresh_tokens_client_id (client_id),
					INDEX idx_oauth_refresh_tokens_user_id (user_id),
					INDEX idx_oauth_refresh_tokens_session_id (session_id),
					INDEX idx_oauth_refresh_tokens_expires_at (expires_at)
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
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS oauth_flow_states;`,
				`DROP TABLE IF EXISTS oauth_refresh_tokens;`,
				`DROP TABLE IF EXISTS oauth_access_grants;`,
				`DROP TABLE IF EXISTS oauth_consents;`,
				`DROP TABLE IF EXISTS oauth_consent_requests;`,
				`DROP TABLE IF EXISTS oauth_auth_requests;`,
				`DROP TABLE IF EXISTS oauth_clients;`,
			)
		},
	}
}
