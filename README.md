# Vaulthook

Vaulthook is a self-hostable webhook gateway that sits between providers (Stripe, GitHub, Shopify, etc.) and your app. It
verifies every inbound webhook's signature, deduplicates and logs it, then forwards it to your destination with retries,
replay, and rate limiting automatically handled for you.

## Why self-hosted

Vaulthook runs entirely on your own server. The payloads, signing secrets, and delivery history never leave your
infrastructure. No per-event pricing, no third-party in the request path, and you own the entire system.

Vaulthook's purpose is intentionally kept narrow. Verify, dedupe, log, forward, retry if needed. It doesn't do fan-out,
payload transformation, or alerting. Once an event is delivered, that's your app's job.

## Architecture

Inbound requests hit Caddy (TLS termination) first, which then forwards it to the Go backend that verifies the signature
and dedupes the event before writing it to Postgres. Only after the write succeeds (the outbox) does a queue worker
attempt forwarding, this ensures that a crash or forwarding failure never loses an event, it just retries later. Events
are NOT processed in a first-in, first-out (FIFO) order, meaning that the destination should handle webhooks sent by
this proxy in an idempotent manner.

Queue workers handle first-attempt delivery, a separate retry worker pool handles backoff retries independently, so a
backlog of retries can't block new events from being processed. A dedicated replay worker handles manually replayed
events on its own pool, isolated from live traffic. Each provider destination has its own rate limiter, so a slow or
throttled endpoint only affects deliveries to that destination.

## Features

- **Cryptographic signature verification** - every inbound request is verified against the provider's signing secret
  before any payload touches the database. Invalid requests are rejected immediately.
- **Duplicate event protection** - each event is deduped on `(provider_id, event_id)` before processing, so provider
  retries never get forwarded twice.
- **Outbound rate limiting** - each provider destination has its own token bucket, ensuring outbound requests never
  exceed the user-configured requests-per-second limit for that destination.
- **Full payload logging** - raw headers, body, timestamp, event type, response code, and retry count logged for every
  event.
- **Transactional outbox pattern** - events are committed to the database before any forwarding attempt, guaranteeing
  zero data loss even under load.
- **Manual replay** - replay any past event from the database. A dedicated worker picks it up within seconds.
- **Powerful event search** - instantly find any webhook by ID, or filter by provider, event type, delivery status,
  response code, date range, payload contents, and more across the entire event history.
- **Live SSE dashboard** - real-time event feed pushed from the server. Pause and resume the stream at any time without
  missing updates.
- **AES-256-GCM encrypted secrets** - provider signing secrets are encrypted at rest, a second layer of protection if
  your server is ever compromised.

## Quick Start (local)

```bash
git clone https://github.com/JBK2116/vaulthook
cd vaulthook/deploy
cp ../env_example.txt .env
# Fill in your .env values
docker compose up -d --build
```

Vaulthook is now running at `http://localhost`.

## Production Deployment

Vaulthook ships as a single Docker image behind Caddy. All config lives in your `.env`. The `deploy/` folder contains:

- `docker-compose.yml` - Vaulthook + PostgreSQL + Caddy
- `Caddyfile` - TLS termination and reverse proxy config
- `entrypoint.sh` - runs database migrations then starts the binary

Replace `:80` in `deploy/Caddyfile` with your actual domain (e.g. `example.com`) and Caddy will automatically request a
Let's Encrypt TLS certificate. Then point your providers at `https://yourdomain.com/api/webhooks/:provider`.

## Environment Variables

See `env_example.txt` for the full reference. Key variables:

| Variable                 | Purpose                                     |
|--------------------------|---------------------------------------------|
| `ADMIN_EMAIL`            | Dashboard login email                       |
| `ADMIN_PASSWORD`         | Dashboard login password                    |
| `JWT_SECRET`             | JWT signing secret                          |
| `MASTER_KEY`             | AES-256 key for encrypting provider secrets |
| `DATABASE_URL`           | PostgreSQL connection string                |
| `REDIS_URL`              | Redis connection string                     |
| `RETRY_INTERVAL_SECONDS` | Backoff interval between retries            |

## Tech Stack

| Layer            | Technology                          |
|------------------|-------------------------------------|
| Backend          | Go                                  |
| Frontend         | SvelteKit, TypeScript, Tailwind CSS |
| Database         | PostgreSQL                          |
| Cache            | Redis                               |
| Migrations       | Goose (Go)                          |
| Auth             | JWT + Refresh Tokens                |
| Encryption       | AES-256-GCM                         |
| Reverse Proxy    | Caddy (automatic TLS)               |
| Containerization | Docker + Docker Compose             |
| Build            | Makefile                            |

## License

MIT
