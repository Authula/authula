-- Rollback initial schema for MySQL

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS key_value_store;
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS auth_settings;
