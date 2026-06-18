-- +goose Up
ALTER TABLE organization_invites ADD COLUMN declined_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE organization_invites DROP COLUMN declined_at;
