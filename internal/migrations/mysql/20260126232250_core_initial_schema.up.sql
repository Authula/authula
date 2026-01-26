SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- USERS

CREATE TABLE IF NOT EXISTS users (
  id BINARY(16) NOT NULL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  email_verified TINYINT(1) DEFAULT 0,
  image LONGTEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_users_email ON users(email);

-- ACCOUNTS

CREATE TABLE IF NOT EXISTS accounts (
  id BINARY(16) NOT NULL PRIMARY KEY,
  user_id BINARY(16) NOT NULL,
  account_id VARCHAR(255) NOT NULL,
  provider_id VARCHAR(255) NOT NULL,
  access_token LONGTEXT,
  refresh_token LONGTEXT,
  id_token LONGTEXT,
  access_token_expires_at TIMESTAMP NULL,
  refresh_token_expires_at TIMESTAMP NULL,
  scope TEXT,
  password TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_accounts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT unique_provider_account UNIQUE(account_id, provider_id),
  INDEX idx_accounts_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SESSIONS

CREATE TABLE IF NOT EXISTS sessions (
  id BINARY(16) NOT NULL PRIMARY KEY,
  user_id BINARY(16) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  token VARCHAR(255) UNIQUE NOT NULL,
  ip_address VARCHAR(45),
  user_agent TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE INDEX idx_sessions_token (token),
  INDEX idx_sessions_user_id (user_id),
  INDEX idx_sessions_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- VERIFICATIONS

CREATE TABLE IF NOT EXISTS verifications (
  id BINARY(16) NOT NULL PRIMARY KEY,
  user_id BINARY(16) NULL,
  identifier VARCHAR(255) NOT NULL,
  token VARCHAR(255) UNIQUE NOT NULL,
  type VARCHAR(50) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_verifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_verifications_user_id (user_id),
  INDEX idx_verifications_identifier (identifier),
  INDEX idx_verifications_token (token),
  INDEX idx_verifications_type (type),
  INDEX idx_verifications_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MySQL Procedures

DROP PROCEDURE IF EXISTS cleanup_sessions;
DROP PROCEDURE IF EXISTS cleanup_verifications;

DELIMITER $$

-- Procedure for Sessions cleanup
CREATE PROCEDURE cleanup_sessions()
BEGIN
  DELETE FROM sessions WHERE expires_at < NOW();
END$$

-- Procedure for Verifications cleanup
CREATE PROCEDURE cleanup_verifications()
BEGIN
  DELETE FROM verifications WHERE expires_at < NOW();
END$$

DELIMITER ;

DELIMITER $$

CREATE PROCEDURE cleanup_all_expired()
BEGIN
  CALL cleanup_sessions();
  CALL cleanup_verifications();
END$$

DELIMITER ;
