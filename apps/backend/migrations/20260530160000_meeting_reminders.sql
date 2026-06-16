
CREATE TABLE meeting_reminders (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    telegram_id BIGINT NOT NULL,
    offset_minutes INT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (meeting_id, telegram_id, offset_minutes)
);

DROP TABLE IF EXISTS meeting_reminders;
