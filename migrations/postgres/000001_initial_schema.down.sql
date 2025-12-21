-- Rollback initial schema for PostgreSQL

-- Drop functions
DROP FUNCTION IF EXISTS cleanup_expired_records();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS key_value_store;
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS auth_settings;

-- Drop extension (be careful - this might affect other schemas)
-- DROP EXTENSION IF EXISTS pgcrypto;
