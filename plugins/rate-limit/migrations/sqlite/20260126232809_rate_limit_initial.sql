-- +goose Up

-- SQLite Rate Limit Plugin Schema
-- Note: This table is temporary/in-memory and ephemeral

PRAGMA temp_store = MEMORY;

CREATE TEMP TABLE IF NOT EXISTS rate_limits (
  key TEXT PRIMARY KEY,
  count INTEGER NOT NULL,
  expires_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at ON rate_limits(expires_at);
