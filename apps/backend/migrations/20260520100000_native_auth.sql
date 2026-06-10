-- +goose Up
ALTER TABLE platform_users ADD COLUMN IF NOT EXISTS phone TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS platform_users_phone_unique ON platform_users (phone) WHERE phone IS NOT NULL AND phone <> '';

CREATE TABLE auth_passkeys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    device_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX auth_passkeys_user_id_idx ON auth_passkeys (user_id);

-- +goose Down
DROP TABLE IF EXISTS auth_passkeys;
DROP INDEX IF EXISTS platform_users_phone_unique;
ALTER TABLE platform_users DROP COLUMN IF EXISTS phone;
