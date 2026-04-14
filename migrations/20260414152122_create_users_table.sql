-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- User ID Generated In Database
    email      VARCHAR(254) NOT NULL UNIQUE, -- User Email Stored In Database
    password   VARCHAR(255) NOT NULL, -- User Password Hash Stored In Database
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP -- Created At Timestamp Generated In Database
);

-- +goose Down
DROP TABLE IF EXISTS users;
