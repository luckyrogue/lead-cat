-- +goose Up
-- user_id is nullable (Telegram-only members have NULL); Postgres treats NULLs as
-- distinct, so multiple NULL-user rows are still allowed. This constraint backs the
-- ON CONFLICT (organization_id, user_id) upsert in AcceptInvitesForEmail.
ALTER TABLE organization_members
  ADD CONSTRAINT organization_members_org_user_unique UNIQUE (organization_id, user_id);
-- +goose Down
ALTER TABLE organization_members
  DROP CONSTRAINT IF EXISTS organization_members_org_user_unique;
