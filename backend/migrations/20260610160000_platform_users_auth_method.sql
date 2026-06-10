-- +goose Up
ALTER TABLE platform_users ADD COLUMN auth_method TEXT NOT NULL DEFAULT '';
-- +goose Down
ALTER TABLE platform_users DROP COLUMN auth_method;
