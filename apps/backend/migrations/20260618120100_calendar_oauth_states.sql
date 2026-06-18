-- +goose Up
CREATE TABLE calendar_oauth_states (
    state      TEXT        PRIMARY KEY,
    email      CITEXT      NOT NULL,
    provider   TEXT        NOT NULL,
    verifier   TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE calendar_oauth_states;
