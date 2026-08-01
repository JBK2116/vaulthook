-- +goose Up
CREATE TABLE IF NOT EXISTS providers (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(), -- Provider ID
    name                 VARCHAR     NOT NULL UNIQUE,                       -- Provider Name
    signing_secret       VARCHAR     NOT NULL,                              -- Provider Signing Secret
    destination_url      VARCHAR     NOT NULL,                              -- Provider Destination URL
    max_retries          INTEGER     NOT NULL,                              -- Provider Max Retry Count Per Webhook
    max_req_second       INTEGER     NOT NULL,                              -- Provider Max Requests Allowed Per Second
    is_configured        BOOLEAN     NOT NULL DEFAULT FALSE,                -- Configured Flag
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()                 -- Created At
);
-- +goose Down
DROP TABLE IF EXISTS providers;
