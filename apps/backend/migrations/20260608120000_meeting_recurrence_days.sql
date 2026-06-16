-- +goose Up
ALTER TABLE meetings ADD COLUMN recurrence_days JSONB;

-- +goose Down
ALTER TABLE meetings DROP COLUMN IF EXISTS recurrence_days;
