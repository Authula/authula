-- SQLite Config Manager Plugin Schema

CREATE TABLE IF NOT EXISTS auth_settings (
  config_version INTEGER PRIMARY KEY NOT NULL DEFAULT 1,
  key VARCHAR(255) NOT NULL UNIQUE,
  value TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auth_settings_config_version ON auth_settings(config_version);
