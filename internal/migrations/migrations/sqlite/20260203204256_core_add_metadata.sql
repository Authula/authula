-- migrate:up

ALTER TABLE users ADD COLUMN metadata JSON NOT NULL DEFAULT '{}';

-- migrate:down

ALTER TABLE users DROP COLUMN metadata;
