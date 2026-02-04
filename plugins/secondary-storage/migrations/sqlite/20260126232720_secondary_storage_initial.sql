-- +goose Up

-- SQLite Secondary Storage Plugin Schema

CREATE TABLE IF NOT EXISTS key_value_store (
  key VARCHAR(255) PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_key_value_store_expires_at ON key_value_store(expires_at);

-- +goose Down

DROP TABLE IF EXISTS key_value_store;
