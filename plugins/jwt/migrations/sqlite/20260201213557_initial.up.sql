-- create "jwks" table
CREATE TABLE `jwks` (
  `id` varchar NOT NULL,
  `public_key` varchar NOT NULL,
  `private_key` varchar NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `expires_at` timestamp NULL,
  PRIMARY KEY (`id`)
);
-- create "refresh_tokens" table
CREATE TABLE `refresh_tokens` (
  `id` varchar NOT NULL,
  `session_id` varchar NOT NULL,
  `token_hash` varchar NOT NULL,
  `expires_at` timestamp NOT NULL,
  `is_revoked` boolean NOT NULL DEFAULT false,
  `revoked_at` timestamp NULL,
  `last_reuse_attempt` timestamp NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  PRIMARY KEY (`id`)
);
-- create index "refresh_tokens_token_hash" to table: "refresh_tokens"
CREATE UNIQUE INDEX `refresh_tokens_token_hash` ON `refresh_tokens` (`token_hash`);
