-- +goose Up
CREATE TYPE delivery_status AS ENUM ('queued', 'processing', 'delivered', 'retrying', 'failed', 'replaying');

CREATE TABLE IF NOT EXISTS webhook_events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Webhook ID Generated In Database
    provider_id       UUID NOT NULL REFERENCES providers(id), -- Provider Associated With Webhook
    provider          VARCHAR NOT NULL, -- Name of the Provider associated with the webhook
    event_id          VARCHAR, -- Provider Event ID Recieved In Request
    event_type        VARCHAR NOT NULL, -- Event Type Received In Request
    headers           JSONB NOT NULL, -- Headers Received In Request 
    payload           JSONB NOT NULL, -- Payload Received In Request
    delivery_status   delivery_status NOT NULL DEFAULT 'queued',
    forwarded_to      VARCHAR NOT NULL, -- Destination Address To Forward Webhook 
    response_code     INT, -- Response Code Of Forwarding Action
    retry_count       INT NOT NULL DEFAULT 0, -- Total Number Of Retries In Forwarding
    next_retry_at     TIMESTAMP, -- Next Retry Attempt Timestamp
    last_error        VARCHAR, -- Last Error Message
    received_at       TIMESTAMP NOT NULL, -- Recieved At Timestamp Of Webhook Event
    created_at        TIMESTAMP NOT NULL DEFAULT NOW(), -- Created At Timestamp Generated In Database
    updated_at        TIMESTAMP NOT NULL DEFAULT NOW() -- Last Updated Timestamp Generated In Database
);

-- automatically updates updated_at whenever the row is modified

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_set_updated_at
BEFORE UPDATE ON webhook_events
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_set_updated_at ON webhook_events;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS webhook_events;
DROP TYPE IF EXISTS delivery_status;
