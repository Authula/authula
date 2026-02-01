-- create "rate_limits" table
CREATE TABLE `rate_limits` (
  `key` varchar NOT NULL,
  `count` integer NULL,
  `expires_at` timestamp NULL,
  PRIMARY KEY (`key`)
);
