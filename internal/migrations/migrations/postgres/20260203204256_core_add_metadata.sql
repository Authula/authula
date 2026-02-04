-- migrate:up

ALTER TABLE users ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::JSONB;

-- migrate:down

ALTER TABLE users DROP COLUMN IF EXISTS metadata;
