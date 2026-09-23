# Lison Backend

Go API for the Lison Telegram Mini App.

## Local checks

```bash
go test ./...
go run ./cmd/contentlint
```

The API runs on port `8080` by default and applies embedded PostgreSQL migrations on startup.

## Voice assistant (Макс)

Maks is a spoken Russian tutor built on the Gemini Live API. Audio does **not**
pass through this API: the Mini App opens a WebSocket straight to Google and the
backend only mints a short-lived token for it, so nginx and Go timeouts stay out
of the audio path.

Set `GEMINI_API_KEY` to switch it on. `GEMINI_LIVE_MODEL` (default
`gemini-3.8-live`) and `ASSISTANT_DAILY_MINUTES` (default 120) tune the model and
the per-learner daily quota. Without an API key the assistant reports itself as
unavailable and nothing else changes.

The persona, voice and transcription settings are pinned into the token so the
browser cannot rewrite them; the tools and the session resumption handle are
deliberately left free, because the client has to set those per connection. The
browser translates each tool call into an ordinary authenticated request to this
API, so the assistant reaches exactly what the learner already can.

Spent minutes are reported by the client and are therefore spoofable. What
actually bounds the spend is the daily cap on minted tokens, since each token is
good for one connection only.

## Docker

```bash
docker build -t lison-backend .
```

Pushes to `main` run tests, publish `ghcr.io/xbakhrom/lison-backend` and update the production backend container. See [deploy/README.md](deploy/README.md) for server preparation.
