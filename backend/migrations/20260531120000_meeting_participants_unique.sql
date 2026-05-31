-- +goose Up
ALTER TABLE meeting_participants ADD CONSTRAINT meeting_participants_unique UNIQUE (meeting_id, email);

-- +goose Down
ALTER TABLE meeting_participants DROP CONSTRAINT IF EXISTS meeting_participants_unique;
