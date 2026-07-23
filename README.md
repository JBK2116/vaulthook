## Features

- **Cryptographic signature verification** - every inbound request is verified against the provider's signing secret before any payload touches the database. Invalid requests are rejected immediately.
- **Full payload logging** - raw headers, body, timestamp, event type, response code, and retry count logged for every event.
- **Transactional outbox pattern** - events are committed to the database before any forwarding attempt, guaranteeing zero data loss even under load.
- **Background worker pool** - concurrent queue workers, retry workers with exponential backoff, and a dedicated replay worker all run independently, configurable via `.env`.
- **Manual replay** - replay any past event from the dashboard. A dedicated worker picks it up within seconds.
- **Powerful event search** - instantly find any webhook by ID, or filter by provider, event type, delivery status, response code, date range, payload contents, and retry/error state, with infinite scroll across your entire event history.
- **Live SSE dashboard** - real-time event feed pushed from the server. Pause and resume the stream at any time without missing updates.
- **AES-256-GCM encrypted secrets** - provider signing secrets are encrypted at rest. A second layer of protection if your server is ever compromised.
- **Fully configurable** - worker counts, retry intervals, max retries, and more via a single `.env` file.

---

## Quick Start

```bash
git clone https://github.com/JBK2116/vaulthook
cd vaulthook/deploy
cp ../env_example.txt .env
# Fill in your .env values
docker compose up -d --build
```

Vaulthook is now running at `http://localhost`.

**For production**, replace `:80` in `deploy/Caddyfile` with your domain (e.g. `example.com`) and Caddy will automatically request a Let's Encrypt TLS certificate. Then point your providers at `https://yourdomain.com/api/webhooks/:provider`.

---

## Environment Variables

See `env_example.txt` for the full reference. Key variables:

| Variable                 | Purpose                                           |
| ------------------------ | ------------------------------------------------- |
| `ADMIN_EMAIL`            | Dashboard login email                             |
| `ADMIN_PASSWORD`         | Dashboard login password                          |
| `JWT_SECRET`             | JWT signing secret                                |
| `MASTER_KEY`             | AES-256 key for encrypting provider secrets       |
| `DATABASE_URL`           | PostgreSQL connection string                      |
| `TOTAL_QUEUE_WORKERS`    | Number of concurrent queue workers                |
| `TOTAL_RETRY_WORKERS`    | Number of concurrent retry workers                |
| `MAX_RETRIES`            | Max retry attempts before dead-lettering an event |
| `RETRY_INTERVAL_SECONDS` | Backoff interval between retries                  |

---

## Testing

Tested with k6 across multiple runs, including a ramping arrival rate scenario and a concurrent burst spike scenario, run simultaneously. Payloads randomized between 0–8KB to stress DB storage and JSON parsing.

| Mode                   | p(95) Latency | Error Rate  | Result |
| ---------------------- | ------------- | ----------- | ------ |
| Success (baseline)     | 60ms          | 0%          | ✅     |
| Chaos (TCP drops)      | 115ms         | 0%          | ✅     |
| Max load (~1000 req/s) | 132ms         | 0% (server) | ✅     |

**858,000+ events processed in a single max-load run, sustaining ~1000 req/s with zero application-level (5xx) errors.**

Load tests are in `internal/tests/load/`.

For testing Stripe endpoints locally, follow the official guide: [HERE](https://docs.stripe.com/stripe-cli/use-cli)

For testing GitHub endpoints locally, follow the official guide: [HERE](https://docs.github.com/en/webhooks/using-webhooks/handling-webhook-deliveries)

---

## Tech Stack

| Layer            | Technology                          |
| ---------------- | ----------------------------------- |
| Backend          | Go                                  |
| Frontend         | SvelteKit, TypeScript, Tailwind CSS |
| Database         | PostgreSQL                          |
| Migrations       | Goose                               |
| Auth             | JWT + Refresh Tokens                |
| Encryption       | AES-256-GCM                         |
| Reverse Proxy    | Caddy (automatic TLS)               |
| Containerization | Docker + Docker Compose             |
| Build            | Makefile                            |
| Load Testing     | k6                                  |

---

## Deployment

Vaulthook ships as a single Docker image behind Caddy. All config lives in your `.env`. The `deploy/` folder contains:

- `docker-compose.yml` - Vaulthook + PostgreSQL + Caddy
- `Caddyfile` - TLS termination and reverse proxy config
- `entrypoint.sh` - runs migrations then starts the binary

**For production**, replace `:80` in `deploy/Caddyfile` with your actual domain, and Caddy handles automatic TLS via Let's Encrypt.

---

## License

MIT
