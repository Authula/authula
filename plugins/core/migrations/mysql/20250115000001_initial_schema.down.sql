-- Drop procedures
DROP PROCEDURE IF EXISTS cleanup_all_expired;
DROP PROCEDURE IF EXISTS cleanup_verifications;
DROP PROCEDURE IF EXISTS cleanup_sessions;

-- Drop tables
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;
