-- create "key_value_store" table
CREATE TABLE `key_value_store` (
  `key` varchar NOT NULL,
  `value` varchar NULL,
  `expires_at` timestamp NULL,
  `created_at` timestamp NOT NULL DEFAULT (current_timestamp),
  `updated_at` timestamp NOT NULL DEFAULT (current_timestamp),
  PRIMARY KEY (`key`)
);
