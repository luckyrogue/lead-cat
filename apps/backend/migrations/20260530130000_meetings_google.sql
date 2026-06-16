-- +goose Up
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS google_sa_json_enc BYTEA;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS google_subject TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS google_calendar_id TEXT NOT NULL DEFAULT 'primary';

-- +goose Down
ALTER TABLE workspaces DROP COLUMN IF EXISTS google_calendar_id;
ALTER TABLE workspaces DROP COLUMN IF EXISTS google_subject;
ALTER TABLE workspaces DROP COLUMN IF EXISTS google_sa_json_enc;
