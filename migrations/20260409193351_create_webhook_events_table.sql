-- +goose Up
CREATE TABLE IF NOT EXISTS webhook_events (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- Webhook ID Generated In Database
    provider_id       UUID        NOT NULL REFERENCES providers(id),  -- Provider Associated With Webhook
    provider_event_id VARCHAR, -- Provider Event ID Recieved In Request
    event_type        VARCHAR     NOT NULL, -- Event Type Received In Request
    headers           JSONB       NOT NULL, -- Headers Received In Request 
    payload           JSONB       NOT NULL, -- Payload Received In Request
    received_at       TIMESTAMP   NOT NULL, -- Recieved At Timestamp Of Webhook Event
    delivery_status   VARCHAR     NOT NULL, -- Delivery Status Of Webhook Forwarding
    forwarded_to      VARCHAR     NOT NULL, -- Destination Address To Forward Webhook 
    response_code     INT, -- Response Code Of Forwarding Action
    retry_count       INT         NOT NULL DEFAULT 0, -- Total Number Of Retries In Forwarding
    next_retry_at     TIMESTAMP, -- Next Retry Attempt Timestamp
    last_error_message        VARCHAR, -- Last Error Message
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW() -- Created At Timestamp Generated In Database
);

-- +goose Down
DROP TABLE IF EXISTS webhook_events;
