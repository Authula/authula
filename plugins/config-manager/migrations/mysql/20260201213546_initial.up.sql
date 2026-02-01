-- create "auth_settings" table
CREATE TABLE `auth_settings` (
  `config_version` bigint NOT NULL AUTO_INCREMENT,
  `key` varchar(255) NULL,
  `value` json NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`config_version`),
  UNIQUE INDEX `key` (`key`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
