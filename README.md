# Lison Backend

Go API for the Lison Telegram Mini App.

## Local checks

```bash
go test ./...
go run ./cmd/contentlint
```

The API runs on port `8080` by default and applies embedded PostgreSQL migrations on startup.

## Docker

```bash
docker build -t lison-backend .
```

Pushes to `main` run tests, publish `ghcr.io/xbakhrom/lison-backend` and update the production backend container. See [deploy/README.md](deploy/README.md) for server preparation.
