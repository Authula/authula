-- create "jwks" table
CREATE TABLE `jwks` (
  `id` varchar(255) NOT NULL,
  `public_key` varchar(255) NOT NULL,
  `private_key` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expires_at` datetime NULL,
  PRIMARY KEY (`id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "refresh_tokens" table
CREATE TABLE `refresh_tokens` (
  `id` varchar(255) NOT NULL,
  `session_id` varchar(255) NOT NULL,
  `token_hash` varchar(255) NOT NULL,
  `expires_at` datetime NOT NULL,
  `is_revoked` bool NOT NULL DEFAULT 0,
  `revoked_at` datetime NULL,
  `last_reuse_attempt` datetime NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `token_hash` (`token_hash`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
