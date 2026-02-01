-- create "rate_limits" table
CREATE TABLE `rate_limits` (
  `key` varchar(255) NOT NULL,
  `count` bigint NULL,
  `expires_at` datetime NULL,
  PRIMARY KEY (`key`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
