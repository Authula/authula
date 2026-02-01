-- reverse: create index "refresh_tokens_token_hash" to table: "refresh_tokens"
DROP INDEX `refresh_tokens_token_hash`;
-- reverse: create "refresh_tokens" table
DROP TABLE `refresh_tokens`;
-- reverse: create "jwks" table
DROP TABLE `jwks`;
