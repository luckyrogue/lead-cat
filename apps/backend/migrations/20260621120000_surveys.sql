-- +goose Up
CREATE TABLE surveys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX surveys_org_idx ON surveys (organization_id);

CREATE TABLE survey_questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id   UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    order_index INT  NOT NULL,
    prompt      TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('single','multi','rating','text')),
    options     TEXT[] NOT NULL DEFAULT '{}',
    rating_max  INT  NOT NULL DEFAULT 5,
    required    BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX survey_questions_survey_idx ON survey_questions (survey_id);

CREATE TABLE survey_responses (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id             UUID NOT NULL REFERENCES surveys(id),
    organization_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    booking_event_type_id UUID REFERENCES booking_event_types(id) ON DELETE SET NULL,
    token                 TEXT NOT NULL UNIQUE,
    booker_email          TEXT NOT NULL DEFAULT '',
    booker_name           TEXT NOT NULL DEFAULT '',
    decline_reason        TEXT NOT NULL DEFAULT '',
    status                TEXT NOT NULL DEFAULT 'sent' CHECK (status IN ('sent','completed')),
    answers               JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at          TIMESTAMPTZ
);
CREATE INDEX survey_responses_survey_idx ON survey_responses (survey_id);
CREATE INDEX survey_responses_org_idx ON survey_responses (organization_id);

ALTER TABLE booking_event_types
    ADD COLUMN survey_id UUID REFERENCES surveys(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE booking_event_types DROP COLUMN survey_id;
DROP TABLE survey_responses;
DROP TABLE survey_questions;
DROP TABLE surveys;
