-- +goose Up
CREATE TABLE booking_event_types (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_user_id       UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    organization_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug               TEXT NOT NULL UNIQUE,
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    duration_mins      INT  NOT NULL,
    active             BOOLEAN NOT NULL DEFAULT true,
    timezone           TEXT NOT NULL DEFAULT '',
    avail_weekdays     INT[] NOT NULL DEFAULT '{1,2,3,4,5}',
    avail_start_minute INT  NOT NULL DEFAULT 540,
    avail_end_minute   INT  NOT NULL DEFAULT 1020,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX booking_event_types_host_idx ON booking_event_types (host_user_id);

-- +goose Down
DROP TABLE booking_event_types;
