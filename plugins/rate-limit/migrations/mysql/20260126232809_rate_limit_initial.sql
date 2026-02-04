-- migrate:up

CREATE TABLE IF NOT EXISTS rate_limits (
  key VARCHAR(255) PRIMARY KEY,
  count INTEGER NOT NULL,
  expires_at TIMESTAMP NOT NULL
) ENGINE=MEMORY;

-- migrate:down

DROP TABLE IF EXISTS rate_limits;
