-- migrate:up

ALTER TABLE users ADD COLUMN metadata JSON;

-- migrate:down

ALTER TABLE users DROP COLUMN metadata;
