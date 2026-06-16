

ALTER TABLE workspaces RENAME TO organizations;
ALTER TABLE workspace_members RENAME TO organization_members;

ALTER TABLE organization_members  RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE employees             RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE meetings              RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE pending_chat_links    RENAME COLUMN workspace_id TO organization_id;

ALTER INDEX workspaces_notify_chat_id_unique RENAME TO organizations_notify_chat_id_unique;

DROP INDEX IF EXISTS workspaces_singleton_idx;

ALTER TABLE organizations
    ADD COLUMN created_by_user_id UUID REFERENCES platform_users(id),
    ADD COLUMN plan TEXT NOT NULL DEFAULT 'free';

ALTER TABLE organizations
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS created_by_user_id;

CREATE UNIQUE INDEX workspaces_singleton_idx ON organizations ((true)) WHERE name = 'Lead Cat';

ALTER INDEX organizations_notify_chat_id_unique RENAME TO workspaces_notify_chat_id_unique;

ALTER TABLE pending_chat_links    RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE meetings              RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE employees             RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE organization_members  RENAME COLUMN organization_id TO workspace_id;

ALTER TABLE organization_members RENAME TO workspace_members;
ALTER TABLE organizations        RENAME TO workspaces;
