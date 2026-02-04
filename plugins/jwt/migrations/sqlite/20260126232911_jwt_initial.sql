-- +goose Up

-- SQLite JWT Plugin Schema

CREATE TABLE IF NOT EXISTS jwks (
  id TEXT PRIMARY KEY,
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_jwks_expires_at ON jwks(expires_at);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  is_revoked INTEGER DEFAULT 0,
  revoked_at TIMESTAMP NULL,
  last_reuse_attempt TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_id ON refresh_tokens(session_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
-- Partial Index (Optimized for SQLite)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked_only 
ON refresh_tokens(is_revoked) 
WHERE is_revoked = 1;

-- +goose Down

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS jwks;
