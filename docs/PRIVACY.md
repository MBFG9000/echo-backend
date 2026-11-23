# Privacy and data security

Echo is built so a person can publish without creating an identity account. This document describes the data the current code stores, the data it refuses to store, and the controls an operator must set. It matches the implementation in this repository.

## Data the service stores

Registration does not take a body. The server creates:

- a random user id
- a pseudonym of the form adjective-noun-number
- a bcrypt hash of the SHA-256 digest of the session token
- timestamps for creation and expiry
- an `is_admin` flag, false for a normal registration

The user table has no name, email, phone number, password, or government identifier.

Posts and replies store the author id, the pseudonym shown on the post, the text (1–280 characters), and counters or timestamps needed for the feed. Reactions store the user id, the post id, and `upvote` or `downvote`. Reports store the reporter id, the post id, a reason (1–500 characters), and the moderation outcome.

Session state is also kept in Redis under a per-user key and expires with the token. The default token lifetime is 24 hours (`JWT_TTL`).

Post text is public. Anyone who can call the feed or `POST /posts/get` can read a post that is not hidden.

## Data the service does not store

The request pipeline deletes `X-Forwarded-For`, `X-Real-Ip`, `Cf-Connecting-Ip`, and `True-Client-Ip` before later handlers run (`internal/middleware/noip.go`). The HTTP logger records method, path, status, latency, and, when a session is present, the pseudonym. It does not record a client address. A development audit panics if those forwarding headers are still present, and `internal/audit/ip_audit_test.go` fails the build if application code calls `ClientIP`.

Rate limiting does not key on an IP address. A request with a bearer token is counted under the SHA-256 of that token. A request without a token is counted under a single anonymous subject.

Echo does not sell data and does not send it to a third-party analytics service. The process talks to PostgreSQL and Redis configured by the operator.

## Security controls

- Protected routes require a bearer JWT. The server checks the signature, then compares the token digest with the bcrypt hash stored for that session.
- Refresh rotates the token. The previous token no longer matches the stored hash.
- Production mode redirects cleartext requests to HTTPS and sets `Strict-Transport-Security`.
- Every response sets `Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, and a locked-down `Permissions-Policy`. The server name and `X-Powered-By` headers are cleared.
- If `ONION_ADDRESS` is set, responses advertise that Tor v3 hostname in `Onion-Location`.
- Create-post and general traffic are rate limited in Redis.
- Admin list and action routes require both a valid session and `is_admin`.
- A post is hidden automatically when the number of distinct reports reaches `MODERATION_AUTO_HIDE_THRESHOLD` (default 3). An admin can dismiss a report, hide the post, or ban the author. A ban hides the post and deletes the author session in Redis.
- The author of a post can delete that post. Replies are removed with the post by the foreign key.

## Operator duties

The repository ships with local defaults that are not a production deployment. Before exposing an instance, the operator must:

- set `JWT_SECRET` to a long random value
- terminate TLS in front of the process and run with `APP_ENV=production`
- set `DB_SSL_MODE` to a verifying mode and keep PostgreSQL and Redis off the public internet
- avoid adding IP addresses in a reverse-proxy access log if the deployment promises anonymity
- restrict who can read database backups, because post text and pseudonyms are personal data even without a legal name
- grant `is_admin` only to moderators who follow [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md)

## Retention and deletion

Session rows and Redis keys expire with `JWT_TTL`. Posts remain until the author deletes them or a moderator hides them. Hidden posts are excluded from public reads; the row is kept so the moderation decision can be audited by an admin. This deployment does not have a separate account-erasure API, because there is no identity account to erase. Deleting a post removes that post and its replies.

## Changes to this policy

A pull request that stores a new field, logs a network address, or sends data to a new service must update this document in the same change. See [CONTRIBUTING.md](../CONTRIBUTING.md).
