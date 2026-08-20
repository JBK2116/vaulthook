-- +goose Up
INSERT INTO providers (name, signing_secret, destination_url, max_retries, max_req_second, is_configured) VALUES
    ('Shopify', '', '', 5, 120, FALSE);
-- +goose Down
DELETE FROM webhook_events WHERE provider_id IN (SELECT id FROM providers WHERE name IN ('Shopify'));
DELETE FROM providers WHERE name IN ('Shopify');
