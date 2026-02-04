-- +goose Up

ALTER TABLE users ADD COLUMN metadata JSON NOT NULL DEFAULT '{}';

-- +goose Down

ALTER TABLE users DROP COLUMN metadata;
