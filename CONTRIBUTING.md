# Contributing to Echo

Echo is an anonymous microblogging API. Contributions are welcome from the student team and from anyone who follows the [Code of Conduct](CODE_OF_CONDUCT.md). By submitting a change you agree that your contribution is licensed under the [MIT License](LICENSE).

## What to read first

- [README.md](README.md) for how to run the service and for the endpoint list.
- [docs/USER_GUIDE.md](docs/USER_GUIDE.md) for the request shapes a client uses.
- [docs/PRIVACY.md](docs/PRIVACY.md) for the data the service is allowed to store.
- [OWNERS.md](OWNERS.md) for who maintains the project.

## Setup

Requirements: Go 1.22 and Docker.

```bash
git clone https://github.com/MBFG9000/echo-backend
cd echo-backend
cp .env.example .env
docker compose up -d
go run ./cmd/server
```

The API listens on http://localhost:8080. Apply schema changes with:

```bash
docker compose run --rm migrate
```

## Making a change

1. Branch from `main`.
2. Keep the existing layout: handlers call services, services call repositories.
3. Do not log client IP addresses, and do not add a name, email, phone, or government identifier to the user record. See [docs/PRIVACY.md](docs/PRIVACY.md).
4. Cover behavior changes with `go test`. The IP audit test must stay green.
5. Open a pull request against `main` and describe the user-visible effect.

## Checks

Continuous integration on `main` runs:

```bash
go vet ./...
staticcheck ./...
go test ./... -race -cover
```

Run the same commands locally before asking for review.

## Review

A maintainer listed in [OWNERS.md](OWNERS.md) reviews pull requests. Changes to authentication, moderation, logging, or stored fields need a privacy note in the pull request, and an update to [docs/PRIVACY.md](docs/PRIVACY.md) when the stored data changes.

## Reporting bugs and conduct issues

Use GitHub issues for defects and for feature requests. Conduct reports follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
