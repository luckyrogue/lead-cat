-- +goose Up

-- 1. Rename the two core tables.
ALTER TABLE workspaces RENAME TO organizations;
ALTER TABLE workspace_members RENAME TO organization_members;

-- 2. Rename workspace_id columns in every table that has one.
ALTER TABLE organization_members  RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE employees             RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE meetings              RENAME COLUMN workspace_id TO organization_id;
ALTER TABLE pending_chat_links    RENAME COLUMN workspace_id TO organization_id;

-- 3. Rename the notify_chat_id unique partial index.
ALTER INDEX workspaces_notify_chat_id_unique RENAME TO organizations_notify_chat_id_unique;

-- 4. Drop the singleton uniqueness constraint (SaaS: multiple orgs are now allowed).
DROP INDEX IF EXISTS workspaces_singleton_idx;

-- 5. Add new columns required by SaaS Phase 0.
ALTER TABLE organizations
    ADD COLUMN created_by_user_id UUID REFERENCES platform_users(id),
    ADD COLUMN plan TEXT NOT NULL DEFAULT 'free';

-- +goose Down

-- Reverse new columns.
ALTER TABLE organizations
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS created_by_user_id;

-- Restore singleton index (must be added BEFORE renaming the table back).
CREATE UNIQUE INDEX workspaces_singleton_idx ON organizations ((true)) WHERE name = 'Lead Cat';

-- Restore index name.
ALTER INDEX organizations_notify_chat_id_unique RENAME TO workspaces_notify_chat_id_unique;

-- Restore workspace_id columns.
ALTER TABLE pending_chat_links    RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE meetings              RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE employees             RENAME COLUMN organization_id TO workspace_id;
ALTER TABLE organization_members  RENAME COLUMN organization_id TO workspace_id;

-- Restore table names.
ALTER TABLE organization_members RENAME TO workspace_members;
ALTER TABLE organizations        RENAME TO workspaces;
