-- +goose Up
ALTER TABLE admin_audit_log DROP CONSTRAINT IF EXISTS admin_audit_log_actor_user_id_fkey;
ALTER TABLE admin_audit_log ALTER COLUMN actor_user_id DROP NOT NULL;
ALTER TABLE admin_audit_log ADD COLUMN actor_kind TEXT NOT NULL DEFAULT 'bot';

-- +goose Down
ALTER TABLE admin_audit_log DROP COLUMN actor_kind;
-- NOTE: re-adding NOT NULL + FK will fail if web-actor rows exist; acceptable for dev rollback.
ALTER TABLE admin_audit_log ALTER COLUMN actor_user_id SET NOT NULL;
ALTER TABLE admin_audit_log ADD CONSTRAINT admin_audit_log_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES bot_users(id);
