-- Drop triggers
DROP TRIGGER IF EXISTS update_verifications_updated_at_trigger ON verifications;
DROP TRIGGER IF EXISTS update_sessions_updated_at_trigger ON sessions;
DROP TRIGGER IF EXISTS update_accounts_updated_at_trigger ON accounts;
DROP TRIGGER IF EXISTS update_users_updated_at_trigger ON users;

-- Drop tables
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;

-- Drop functions
DROP FUNCTION IF EXISTS cleanup_expired_records();
DROP FUNCTION IF EXISTS core_update_updated_at_column_func();
