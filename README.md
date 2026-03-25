# `<echo> backend`

![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)

Anonymous microblogging API. No email. No phone. No PII stored.
Built with Go + Gin + PostgreSQL + Redis.

## Architecture

```mermaid
graph TD
    Client -->|HTTPS / WSS| RateLimiter
    RateLimiter -->|Redis token bucket| Router[Gin Router]
    Router --> Auth[JWT Middleware]
    Auth --> Handlers

    subgraph Handlers
        H1[POST /auth/session]
        H2[POST /posts]
        H3[GET /posts/feed]
        H4[GET /ws/feed]
    end

    Handlers --> Services
    Services --> GORM
    Services --> Redis[(Redis\nsessions · trending)]
    GORM --> Postgres[(PostgreSQL\nusers · posts · threads)]

    style Redis fill:#DC382D,color:#fff
    style Postgres fill:#336791,color:#fff
```

## Quick Start

```bash
git clone https://github.com/YOUR_USERNAME/echo-backend
cd echo-backend
cp .env.example .env
docker compose up -d
go run ./cmd/server
```

API: http://localhost:8080

## Migrations

Versioned SQL migrations are stored in [migrations](migrations).

```bash
docker compose run --rm migrate
```

On `docker compose up -d`, the `migrate` service also runs automatically once PostgreSQL is healthy.

## Endpoints

```
POST  /auth/register             create anonymous session → JWT + pseudonym
POST  /auth/refresh              rotate session token
POST  /posts                     create post (≤280 chars, requires JWT)
GET   /posts/:id                 fetch post by id
DELETE /posts/:id                delete own post
POST  /posts/:id/replies         create reply
GET   /posts/:id/replies         list replies
POST  /posts/:id/react           body: {"kind":"upvote"|"downvote"}
POST  /posts/:id/report          create moderation report
GET   /feed/latest               latest feed with cursor pagination
GET   /feed/trending             trending feed
GET   /ws/feed                   WebSocket — real-time new posts
GET   /admin/reports             list open reports (admin JWT)
POST  /admin/reports/:id/action  body: {"action":"dismiss"|"hide"|"ban","note":"..."}
```

## Project Structure

```
echo-backend/
├── cmd/server/         # main.go entrypoint
├── internal/
│   ├── handler/        # HTTP + WS handlers
│   ├── service/        # business logic
│   ├── repository/     # DB queries (GORM)
│   └── middleware/     # JWT, rate-limit
├── pkg/                # shared utils (pseudonym gen)
├── docker-compose.yml
└── .env.example
```

## Stack

| Layer    | Tech                     |
|----------|--------------------------|
| Language | Go 1.22                  |
| HTTP     | Gin                      |
| ORM      | GORM + PostgreSQL        |
| Cache    | Redis 7                  |
| Auth     | JWT (golang-jwt/jwt/v5)  |
| Realtime | WebSocket (gorilla/ws)   |
| CI/CD    | GitHub Actions           |
