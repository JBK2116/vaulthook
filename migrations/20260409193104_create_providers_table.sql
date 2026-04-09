-- +goose Up
CREATE TABLE IF NOT EXISTS providers (
    id              UUID        PRIMARY KEY,
    name            VARCHAR     NOT NULL UNIQUE,
    signing_secret  VARCHAR     NOT NULL,
    destination_url VARCHAR     NOT NULL,
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS providers;
