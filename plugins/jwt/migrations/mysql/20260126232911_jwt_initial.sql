-- +goose Up

CREATE TABLE IF NOT EXISTS jwks (
  id BINARY(16) NOT NULL PRIMARY KEY,
  public_key TEXT NOT NULL,
  private_key TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NULL
);

CREATE INDEX idx_jwks_expires_at ON jwks(expires_at);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id BINARY(16) NOT NULL PRIMARY KEY,
  session_id BINARY(16) NOT NULL,
  token_hash VARCHAR(64) UNIQUE NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  is_revoked BOOLEAN DEFAULT FALSE,
  revoked_at TIMESTAMP NULL,
  last_reuse_attempt TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_refresh_tokens_session
    FOREIGN KEY (session_id) REFERENCES sessions(id)
    ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_session_id ON refresh_tokens(session_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_active_session ON refresh_tokens(session_id, is_revoked);
CREATE INDEX idx_refresh_tokens_last_reuse_attempt ON refresh_tokens(last_reuse_attempt);

-- Cleanup query for expired tokens
-- MySQL procedure to cleanup expired tokens
DROP PROCEDURE IF EXISTS cleanup_expired_refresh_tokens;

DELIMITER $$

CREATE PROCEDURE cleanup_expired_refresh_tokens()
BEGIN
  DELETE FROM refresh_tokens WHERE expires_at < NOW();
END$$

DELIMITER ;

-- +goose Down

DROP PROCEDURE IF EXISTS cleanup_expired_refresh_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS jwks;
