-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- Token ID Generated In Database
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- User Associated With Token
    token      VARCHAR(255) UNIQUE NOT NULL, -- Token String 
    expires_at TIMESTAMP    NOT NULL, -- Token Expiry Timestamp
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Token Created At Timestamp Generated In Database
    revoked    BOOLEAN      NOT NULL DEFAULT FALSE -- Token Revoked Boolean
);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token); -- Index To Speed Up Token Retrievals

-- +goose Down
DROP INDEX IF EXISTS idx_refresh_tokens_token;
DROP TABLE IF EXISTS refresh_tokens;
