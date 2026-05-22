# echo-backend

Echo backend is an anonymous microblogging API for posts, replies, reactions, realtime feed updates, and moderation workflows. It uses JWT-based anonymous sessions, PostgreSQL for persistence, and Redis for sessions, caching, and rate limiting. The codebase follows handler → service → repository layering for clear boundaries.

## Prerequisites

- Go 1.22
- Docker

## Quick start

```bash
git clone https://github.com/YOUR_USERNAME/echo-backend
cd echo-backend
cp .env.example .env
docker compose up -d
go run ./cmd/server
```

API base URL: http://localhost:8080

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
| GET | /health | No | Service health with DB and Redis checks |
| POST | /auth/register | No | Create anonymous session and return JWT + pseudonym |
| POST | /auth/refresh | Yes | Rotate JWT for current session |
| POST | /posts | Yes | Create post (max 280 chars) |
| GET | /posts/:id | No | Get post by ID |
| GET | /posts/:id/share | No | Get canonical share URL for a post |
| DELETE | /posts/:id | Yes | Delete own post |
| POST | /posts/:id/replies | Yes | Create reply to post |
| GET | /posts/:id/replies | No | List post replies |
| POST | /posts/:id/react | Yes | Upvote or downvote post |
| POST | /posts/:id/report | Yes | Report post for moderation |
| GET | /feed/latest | No | Latest feed with cursor pagination |
| GET | /feed/trending | No | Trending feed |
| GET | /ws/feed | No | Realtime feed websocket |
| GET | /admin/reports | Admin | List open moderation reports |
| POST | /admin/reports/:id/action | Admin | Resolve report with dismiss/hide/ban |

## Migrations

```bash
docker compose run --rm migrate
```
