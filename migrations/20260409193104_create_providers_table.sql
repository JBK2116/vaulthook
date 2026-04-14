-- +goose Up
CREATE TABLE IF NOT EXISTS providers (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- Provider ID
    name            VARCHAR     NOT NULL UNIQUE, -- Provider Name
    signing_secret  VARCHAR     NOT NULL, -- Provider Signing Secret For Authentication
    destination_url VARCHAR     NOT NULL, -- Provider Destination URL To Forward Events 
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW() -- Created At Timestamp Generated In Database
);

-- +goose Down
DROP TABLE IF EXISTS providers;
