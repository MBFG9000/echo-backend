# Echo user guide

Echo is an anonymous microblogging API. A client creates a session, receives a pseudonym and a token, then publishes short posts, replies, and votes. No account form is required.

Base URL for a local run: `http://localhost:8080`.

Start the service with the steps in [README.md](../README.md).

## Create a session

`POST /auth/register` takes an empty body. The response contains a bearer token and a generated pseudonym such as `lunar-fox-482`. Send the token on later requests as `Authorization: Bearer <token>`.

```bash
curl -s -X POST http://localhost:8080/auth/register
```

Refresh a live session with the current token. The previous token stops working after a successful refresh.

```bash
curl -s -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"<token>\"}"
```

Sessions expire. The default lifetime is 24 hours (`JWT_TTL`).

## Read the feed

Latest posts use cursor pagination. Trending posts are ordered by score.

```bash
curl -s -X POST http://localhost:8080/feed/latest \
  -H "Content-Type: application/json" \
  -d "{\"limit\":20}"

curl -s -X POST http://localhost:8080/feed/trending \
  -H "Content-Type: application/json" \
  -d "{\"limit\":20}"
```

Fetch one post or its replies without a token:

```bash
curl -s -X POST http://localhost:8080/posts/get \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"<post-id>\"}"

curl -s -X POST http://localhost:8080/posts/replies/list \
  -H "Content-Type: application/json" \
  -d "{\"postId\":\"<post-id>\",\"limit\":20}"
```

`GET /ws/feed` is a WebSocket of new public posts. `GET /health` reports whether PostgreSQL and Redis respond.

## Write

These routes require the bearer token. Post and reply text must be 1 to 280 characters after trimming.

```bash
curl -s -X POST http://localhost:8080/posts \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"Hello from Echo\"}"

curl -s -X POST http://localhost:8080/posts/replies/create \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"postId\":\"<post-id>\",\"content\":\"A reply\"}"

curl -s -X POST http://localhost:8080/posts/react \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"postId\":\"<post-id>\",\"kind\":\"upvote\"}"
```

`kind` is `upvote` or `downvote`. Delete only a post created by the same session:

```bash
curl -s -X POST http://localhost:8080/posts/delete \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"<post-id>\"}"
```

## Report a post

Any signed-in session can file one report per post. The reason is 1 to 500 characters.

```bash
curl -s -X POST http://localhost:8080/posts/report \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"postId\":\"<post-id>\",\"reason\":\"Harassment\"}"
```

When distinct reporters reach `MODERATION_AUTO_HIDE_THRESHOLD` (default 3), the post is hidden from public reads.

## Moderation

Admin routes require a session whose user is marked admin. Actions are `dismiss`, `hide`, and `ban`. `hide` and `ban` hide the post. `ban` also deletes the author session in Redis.

```bash
curl -s -X POST http://localhost:8080/admin/reports/list \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d "{\"limit\":20,\"offset\":0}"

curl -s -X POST http://localhost:8080/admin/reports/action \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d "{\"reportId\":\"<report-id>\",\"action\":\"hide\",\"note\":\"Breaks the code of conduct\"}"
```

## Privacy expectations for users

The API does not ask who you are. It stores the pseudonym, the post text, and a session secret. Details of what is stored, and what operators must configure, are in [PRIVACY.md](PRIVACY.md).
