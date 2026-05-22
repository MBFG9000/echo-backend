# echo-backend

Echo backend is an anonymous microblogging API for posts, replies, reactions, realtime feed updates, and moderation workflows. It uses JWT-based anonymous sessions, PostgreSQL for persistence, and Redis for sessions, caching, and rate limiting. The codebase follows handler → service → repository layering for clear boundaries.

## Prerequisites

- Go 1.25
- Docker

## Quick start (local `go run`)

Recommended for day-to-day development: DB/Redis in Docker, API on the host.

```bash
git clone https://github.com/MBFG9000/echo-backend
cd echo-backend
cp .env.example.localhost .env
cp docker-compose.override.example.yml docker-compose.override.yml
docker compose up -d postgres redis
docker compose run --rm migrate
go run ./cmd/server
```

API base URL: http://localhost:8080

Optional Swagger UI (expects the server above on :8080):

```bash
docker compose up -d swagger-ui
```

Swagger UI: http://localhost:8081

## Quick start (full Docker stack)

Runs the API in a container instead of `go run`:

```bash
cp .env.example .env
docker compose up -d
```

API base URL: http://localhost:8080  
Swagger UI: http://localhost:8081

Do not run `go run` and `docker compose up app` at the same time — both use port 8080.

Set `APP_PUBLIC_URL` in `.env` to your public frontend origin (no trailing slash). It is used for `GET /posts/:id/share` links and must match the URL users open in the browser (e.g. `http://your-host:8081` on VPS).

## Swagger docs

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.3
swag init -g cmd/server/main.go -o docs
```

## Production-oriented features

| Feature | Endpoint / header | Notes |
|---|---|---|
| Prometheus metrics | `GET /metrics` | `http_requests_total`, latency histogram, `ws_connections_active`, `rate_limit_hits_total` |
| Request tracing | `X-Request-ID` | Echoed in response; included in structured logs |
| Idempotent creates | `Idempotency-Key: <uuid>` | `POST /posts`, `POST /posts/:id/replies` — 24h Redis cache |
| Full-text search | `POST /posts/search` | PostgreSQL `tsvector` + GIN (migration `000005`) |
| Transactional outbox | internal | New posts → `outbox_events` → worker → WebSocket broadcast |
| Feed caching | `GET /feed/latest` | `ETag` / `If-None-Match` → `304` when unchanged |

```bash
curl -s http://localhost:8080/metrics | head
curl -sI -H 'If-None-Match: W/"abc"' 'http://localhost:8080/feed/latest?limit=20'
```

## Architecture (ASCII)

```text
Client
    |
    v
Gin Router
    |
    +--> Middleware (CORS, request log, auth, rate limit)
    |
    v
Handlers
    |
    v
Services
    |
    +--> Redis (sessions, feed cache, rate limit)
    |
    v
Repositories
    |
    v
PostgreSQL
```

## API endpoints
| Method | Path | Auth | Description |
|---|---|---|---|
| GET | /metrics | No | Prometheus metrics |
| GET | /health | No | Service health with DB and Redis checks |
| POST | /auth/register | No | Create anonymous session and return JWT + pseudonym |
| POST | /auth/refresh | No | Rotate JWT (Bearer header or token in body) |
| POST | /posts | Yes | Create post (JSON or multipart with optional file) |
| GET | /posts/:id | No | Get post by ID |
| POST | /posts/get | No | Get post by ID (legacy JSON body: `id`) |
| GET | /posts/:id/share | No | Get canonical share URL for a post |
| GET | /posts/attachments/:id | No | Download post attachment |
| POST | /posts/search | No | Search posts by content or pseudonym |
| DELETE | /posts/:id | Yes | Delete own post |
| POST | /posts/delete | Yes | Delete own post (legacy JSON body: `id`) |
| GET | /posts/:id/replies | No | List post replies |
| POST | /posts/:id/replies | Yes | Create reply to post |
| POST | /posts/replies/create | Yes | Create reply (legacy; optional `parentReplyId`) |
| POST | /posts/replies/list | No | List post replies (legacy JSON body: `postId`) |
| POST | /posts/replies/update | Yes | Edit own reply |
| POST | /posts/replies/delete | Yes | Delete own reply |
| POST | /posts/:id/react | Yes | Upvote or downvote post |
| DELETE | /posts/:id/react | Yes | Remove own reaction on post |
| POST | /posts/replies/:replyId/react | Yes | Upvote or downvote reply |
| DELETE | /posts/replies/:replyId/react | Yes | Remove own reaction on reply |
| POST | /posts/replies/react | Yes | Upvote or downvote reply (legacy JSON body) |
| POST | /posts/react | Yes | Upvote or downvote post (legacy JSON body: `postId`) |
| POST | /posts/:id/report | Yes | Report post for moderation |
| POST | /posts/report | Yes | Report post (legacy JSON body: `postId`) |
| GET | /feed/latest | No | Latest feed with cursor pagination (`ETag` / `304`) |
| POST | /feed/latest | No | Latest feed (legacy JSON body) |
| GET | /feed/trending | No | Trending feed |
| POST | /feed/trending | No | Trending feed (legacy JSON body) |
| GET | /ws/feed | No | Realtime feed websocket |
| POST | /admin/reports/list | Admin | List open moderation reports |
| POST | /admin/reports/action | Admin | Resolve report with dismiss/hide/ban |

## Migrations

```bash
docker compose run --rm migrate
```
