-- +goose Up
ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS reminder_minutes TEXT NOT NULL DEFAULT '15';

-- +goose Down
ALTER TABLE bot_users DROP COLUMN IF EXISTS reminder_minutes;
