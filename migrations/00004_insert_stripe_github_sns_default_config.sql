-- +goose Up
INSERT INTO providers (name, signing_secret, destination_url, is_configured) VALUES
    ('Github', '', '', FALSE),
    ('Stripe', '', '', FALSE),
    ('SNS',    '', '', FALSE);

-- +goose Down
DELETE FROM providers WHERE name IN ('Github', 'Stripe', 'SNS');
