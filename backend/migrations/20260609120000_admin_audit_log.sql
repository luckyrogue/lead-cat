-- +goose Up
CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID NOT NULL REFERENCES bot_users(id),
    actor_telegram_id BIGINT NOT NULL,
    actor_email TEXT NOT NULL,
    action TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    target_id TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_log_created_at_idx ON admin_audit_log (created_at DESC);
CREATE INDEX admin_audit_log_actor_idx ON admin_audit_log (actor_user_id, created_at DESC);
CREATE INDEX admin_audit_log_action_idx ON admin_audit_log (action, created_at DESC);

CREATE UNIQUE INDEX workspaces_singleton_idx ON workspaces ((true)) WHERE name = 'Lead Cat';

-- +goose Down
DROP INDEX IF EXISTS workspaces_singleton_idx;
DROP INDEX IF EXISTS admin_audit_log_action_idx;
DROP INDEX IF EXISTS admin_audit_log_actor_idx;
DROP INDEX IF EXISTS admin_audit_log_created_at_idx;
DROP TABLE IF EXISTS admin_audit_log;
