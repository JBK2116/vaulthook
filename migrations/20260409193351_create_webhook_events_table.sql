-- +goose Up
CREATE TABLE IF NOT EXISTS webhook_events (
    id                UUID        PRIMARY KEY,
    provider_id       UUID        NOT NULL REFERENCES providers(id),
    provider_event_id VARCHAR,
    event_type        VARCHAR     NOT NULL,
    headers           JSONB       NOT NULL,
    payload           JSONB       NOT NULL,
    received_at       TIMESTAMP   NOT NULL,
    delivery_status   VARCHAR     NOT NULL,
    forwarded_to      VARCHAR     NOT NULL,
    response_code     INT,
    retry_count       INT         NOT NULL DEFAULT 0,
    next_retry_at     TIMESTAMP,
    last_error        VARCHAR
);

-- +goose Down
DROP TABLE IF EXISTS webhook_events;
