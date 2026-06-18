-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE TABLE calendar_connections (
    email             CITEXT      NOT NULL,
    provider          TEXT        NOT NULL,
    access_token_enc  BYTEA       NOT NULL,
    refresh_token_enc BYTEA       NOT NULL,
    expiry            TIMESTAMPTZ NOT NULL,
    scopes            TEXT        NOT NULL DEFAULT '',
    connected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (email, provider)
);

-- +goose Down
DROP TABLE calendar_connections;
