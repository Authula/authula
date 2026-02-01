-- reverse: create index "verifications_token" to table: "verifications"
DROP INDEX `verifications_token`;
-- reverse: create "verifications" table
DROP TABLE `verifications`;
-- reverse: create index "sessions_token" to table: "sessions"
DROP INDEX `sessions_token`;
-- reverse: create "sessions" table
DROP TABLE `sessions`;
-- reverse: create index "accounts_account_id_provider_id" to table: "accounts"
DROP INDEX `accounts_account_id_provider_id`;
-- reverse: create "accounts" table
DROP TABLE `accounts`;
-- reverse: create index "users_email" to table: "users"
DROP INDEX `users_email`;
-- reverse: create "users" table
DROP TABLE `users`;
