-- create "users" table
CREATE TABLE `users` (
  `id` varchar NOT NULL,
  `name` varchar NOT NULL,
  `email` varchar NOT NULL,
  `email_verified` boolean NOT NULL DEFAULT false,
  `image` varchar NULL,
  `metadata` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `updated_at` timestamp NOT NULL DEFAULT (current_timestamp),
  PRIMARY KEY (`id`)
);
-- create index "users_email" to table: "users"
CREATE UNIQUE INDEX `users_email` ON `users` (`email`);
-- create "accounts" table
CREATE TABLE `accounts` (
  `id` varchar NOT NULL,
  `user_id` varchar NOT NULL,
  `account_id` varchar NOT NULL,
  `provider_id` varchar NOT NULL,
  `access_token` varchar NULL,
  `refresh_token` varchar NULL,
  `id_token` varchar NULL,
  `access_token_expires_at` timestamp NULL,
  `refresh_token_expires_at` timestamp NULL,
  `scope` varchar NULL,
  `password` varchar NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `updated_at` timestamp NOT NULL DEFAULT (current_timestamp),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "accounts_account_id_provider_id" to table: "accounts"
CREATE UNIQUE INDEX `accounts_account_id_provider_id` ON `accounts` (`account_id`, `provider_id`);
-- create "sessions" table
CREATE TABLE `sessions` (
  `id` varchar NOT NULL,
  `user_id` varchar NOT NULL,
  `token` varchar NOT NULL,
  `expires_at` timestamp NOT NULL,
  `ip_address` varchar NULL,
  `user_agent` varchar NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `updated_at` timestamp NOT NULL DEFAULT (current_timestamp),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "sessions_token" to table: "sessions"
CREATE UNIQUE INDEX `sessions_token` ON `sessions` (`token`);
-- create "verifications" table
CREATE TABLE `verifications` (
  `id` varchar NOT NULL,
  `user_id` varchar NULL,
  `identifier` varchar NOT NULL,
  `token` varchar NOT NULL,
  `type` varchar NOT NULL,
  `expires_at` timestamp NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `updated_at` timestamp NOT NULL DEFAULT (current_timestamp),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "verifications_token" to table: "verifications"
CREATE UNIQUE INDEX `verifications_token` ON `verifications` (`token`);
