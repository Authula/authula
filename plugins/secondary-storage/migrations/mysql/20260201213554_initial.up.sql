-- create "key_value_store" table
CREATE TABLE `key_value_store` (
  `key` varchar(255) NOT NULL,
  `value` varchar(255) NULL,
  `expires_at` datetime NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`key`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
