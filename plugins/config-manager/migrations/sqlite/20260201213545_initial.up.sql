-- create "auth_settings" table
CREATE TABLE `auth_settings` (
  `config_version` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `key` varchar NULL,
  `value` json NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `updated_at` timestamp NOT NULL DEFAULT (current_timestamp)
);
-- create index "auth_settings_key" to table: "auth_settings"
CREATE UNIQUE INDEX `auth_settings_key` ON `auth_settings` (`key`);
