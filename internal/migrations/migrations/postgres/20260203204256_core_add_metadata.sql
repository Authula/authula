-- +goose Up

ALTER TABLE users ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::JSONB;

-- +goose Down

ALTER TABLE users DROP COLUMN metadata;
