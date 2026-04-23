-- +goose Up
CREATE TABLE IF NOT EXISTS providers (
    id              UUID      PRIMARY KEY DEFAULT gen_random_uuid(), -- Provider ID
    name            VARCHAR   NOT NULL UNIQUE,                       -- Provider Name
    signing_secret  VARCHAR   NOT NULL,                              -- Provider Signing Secret
    destination_url VARCHAR   NOT NULL,                              -- Provider Destination URL
    is_configured   BOOLEAN   NOT NULL DEFAULT FALSE,                -- Configured Flag
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()                 -- Created At
);
-- +goose Down
DROP TABLE IF EXISTS providers;
