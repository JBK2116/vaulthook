-- +goose Up
INSERT INTO providers (name, signing_secret, destination_url, max_retries, max_req_second, is_configured) VALUES
    ('Github', '', '', 5, 100, FALSE),
    ('Stripe', '', '', 5, 90, FALSE);
-- +goose Down
DELETE FROM webhook_events WHERE provider_id IN (SELECT id FROM providers WHERE name IN ('Github', 'Stripe'));
DELETE FROM providers WHERE name IN ('Github', 'Stripe');
