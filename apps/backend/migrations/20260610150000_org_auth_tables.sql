
ALTER TABLE organization_members ADD COLUMN invited_email TEXT;
ALTER TABLE organization_members ALTER COLUMN role SET DEFAULT 'member';
UPDATE organization_members SET role = 'member' WHERE role NOT IN ('owner','admin');

CREATE TABLE organization_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    token_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_by_user_id UUID REFERENCES platform_users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX organization_invites_email_idx ON organization_invites (lower(email)) WHERE accepted_at IS NULL;

CREATE TABLE magic_link_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    token_hash BYTEA NOT NULL,
    purpose TEXT NOT NULL DEFAULT 'login',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX magic_link_tokens_hash_idx ON magic_link_tokens (token_hash);

CREATE TABLE web_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash BYTEA NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT ''
);
CREATE INDEX web_sessions_user_idx ON web_sessions (user_id);

DROP TABLE IF EXISTS web_sessions;
DROP TABLE IF EXISTS magic_link_tokens;
DROP TABLE IF EXISTS organization_invites;
ALTER TABLE organization_members DROP COLUMN invited_email;
