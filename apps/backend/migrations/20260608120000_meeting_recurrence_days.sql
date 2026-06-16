
ALTER TABLE meetings ADD COLUMN recurrence_days JSONB;

ALTER TABLE meetings DROP COLUMN IF EXISTS recurrence_days;
