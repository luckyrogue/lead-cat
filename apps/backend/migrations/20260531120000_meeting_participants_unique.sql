
ALTER TABLE meeting_participants ADD CONSTRAINT meeting_participants_unique UNIQUE (meeting_id, email);

ALTER TABLE meeting_participants DROP CONSTRAINT IF EXISTS meeting_participants_unique;
