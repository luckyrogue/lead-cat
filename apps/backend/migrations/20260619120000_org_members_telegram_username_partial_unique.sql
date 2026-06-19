-- +goose Up
ALTER TABLE organization_members
  DROP CONSTRAINT IF EXISTS workspace_members_workspace_id_telegram_username_key;
CREATE UNIQUE INDEX IF NOT EXISTS organization_members_org_telegram_username_idx
  ON organization_members (organization_id, telegram_username)
  WHERE telegram_username <> '';

-- +goose Down
DROP INDEX IF EXISTS organization_members_org_telegram_username_idx;
ALTER TABLE organization_members
  ADD CONSTRAINT workspace_members_workspace_id_telegram_username_key
  UNIQUE (organization_id, telegram_username);
