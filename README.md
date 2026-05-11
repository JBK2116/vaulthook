# Vaulthook

**A self-hostable webhook gateway. Own your data, inspect everything, replay anything.**

![HookVault Demo](./demo.gif)

---

## What is it?

HookVault sits between your webhook providers and your application. It receives every inbound webhook, cryptographically verifies the signature, logs the full payload and headers to your database, and forwards it to your destination. If delivery fails, background workers retry automatically. If you need to re-send an event, replay it from the dashboard in one click.

No managed service. No third party sees your data. Deploy it on your VPS, point your providers at it, and log in from anywhere.

## Supported Providers

- Stripe

- Github [**Coming Soon**]

- SNS [**Coming Soon**]

---

## Architecture

```
[Stripe / GitHub / AWS SNS]
          │
          ▼
    [Caddy — TLS]
     ├── /* → SvelteKit frontend
     └── /api/* → Go binary :8080
          │
          ▼
  [HookVault — Go :8080]
    ├── /api/webhooks/:provider   ← Inbound receiver
    └── /api/*                   ← Dashboard API (JWT protected)
          │
          ▼
     [PostgreSQL]
          │
          ▼
  [Destination URL — your app]
```

```
hookvault/
├── cmd/
│   ├── api/              ← Entry point
│   └── api-mock/         ← Stripe simulator for testing
├── internal/
│   ├── api/handler/      ← HTTP handlers (auth, events, providers, stripe)
│   ├── auth/             ← JWT + refresh token logic
│   ├── config/           ← Env config, logger, DB setup
│   ├── crypto/           ← AES-256-GCM encryption for signing secrets
│   ├── events/           ← SSE pipeline + event service
│   ├── model/            ← Domain models
│   ├── providers/        ← Provider management
│   ├── tests/load/       ← k6 stress tests
│   └── worker/           ← Queue, retry, replay, and cleanup workers
├── migrations/           ← Goose SQL migrations
├── frontend/             ← SvelteKit dashboard
└── deploy/               ← Dockerfile, docker-compose.yml, Caddyfile
```

---

## Features

- **Cryptographic signature verification** - every inbound request is verified against the provider's signing secret before any payload touches the database. Invalid requests are rejected immediately.
- **Full payload logging** - raw headers, body, timestamp, event type, response code, and retry count logged for every event.
- **Transactional outbox pattern** - events are committed to the database before any forwarding attempt, guaranteeing zero data loss even under load.
- **Background worker pool** - concurrent queue workers, retry workers with exponential backoff, and a dedicated replay worker all run independently, configurable via `.env`.
- **Manual replay** - replay any past event from the dashboard. A dedicated worker picks it up within seconds.
- **Live SSE dashboard** - real-time event feed pushed from the server. Pause and resume the stream at any time without missing updates.
- **AES-256-GCM encrypted secrets** - provider signing secrets are encrypted at rest. A second layer of protection if your server is ever compromised.
- **Fully configurable** - worker counts, retry intervals, max retries, and more via a single `.env` file.

---

## Quick Start

```bash
git clone https://github.com/JBK2116/hookvault
cd hookvault/deploy
cp ../env_example.txt .env
# Fill in your .env values
add your domain into Caddyfile
docker compose up -d
```
HookVault is running behind Caddy with automatic TLS. 
Point your providers at `https://yourdomain.com/api/webhooks/:provider`.

---

## Environment Variables

See `env_example.txt` for the full reference. Key variables:

| Variable | Purpose |
|---|---|
| `ADMIN_EMAIL` | Dashboard login email |
| `ADMIN_PASSWORD` | Dashboard login password |
| `JWT_SECRET` | JWT signing secret |
| `MASTER_KEY` | AES-256 key for encrypting provider secrets |
| `DATABASE_URL` | PostgreSQL connection string |
| `TOTAL_QUEUE_WORKERS` | Number of concurrent queue workers |
| `TOTAL_RETRY_WORKERS` | Number of concurrent retry workers |
| `MAX_RETRIES` | Max retry attempts before dead-lettering |
| `RETRY_INTERVAL_SECONDS` | Backoff interval between retries |

---

## Stress Test Results

Tested with k6 under two scenarios running simultaneously — a ramping arrival rate (up to 600 req/s) and a concurrent burst spike of 100 VUs. Payloads randomized between 0–8KB to stress DB storage and JSON parsing.

| Mode | p(95) Latency | Error Rate | Result |
|---|---|---|---|
| Success (baseline) | 72ms | 0% | ✅ |
| Chaos (TCP drops) | 115ms | 0% | ✅ |

**195,000+ total events processed. Zero data loss across all runs.**

Load tests are in `internal/tests/load/`.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go |
| Frontend | SvelteKit, TypeScript, Tailwind CSS |
| Database | PostgreSQL |
| Migrations | Goose |
| Auth | JWT + Refresh Tokens |
| Encryption | AES-256-GCM |
| Reverse Proxy | Caddy (automatic TLS) |
| Containerization | Docker + Docker Compose |
| Build | Makefile |
| Load Testing | k6 |

---

## Deployment

HookVault ships as a single Docker image behind Caddy. All config lives in your `.env`. The `deploy/` folder contains:

- `docker-compose.yml` - HookVault + PostgreSQL + Caddy
- `Caddyfile` - TLS termination and reverse proxy config
- `entrypoint.sh` - runs migrations then starts the binary

Point your domain at the server, set `DOMAIN` in your `.env`, and Caddy handles the rest.

---

## License

MIT
