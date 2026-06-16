-- +goose Up
ALTER TABLE meetings ADD COLUMN series_id UUID;
CREATE INDEX meetings_series_idx ON meetings (series_id);

-- +goose Down
DROP INDEX IF EXISTS meetings_series_idx;
ALTER TABLE meetings DROP COLUMN IF EXISTS series_id;
