-- +goose Up

ALTER TABLE users ADD COLUMN metadata JSON;

-- +goose Down

ALTER TABLE users DROP COLUMN metadata;
