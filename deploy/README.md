# Docker production deployment

The backend and webapp repositories publish images to GitHub Container Registry. Both workflows update their own service in one server-side Docker Compose project.

## GitHub configuration

Create a `production` environment in both repositories and add:

Secrets:

- `SSH_HOST`
- `SSH_USER`
- `SSH_PRIVATE_KEY`
- `SSH_KNOWN_HOSTS` — a verified host-key line from `ssh-keyscan -H <host>`

Optional variables:

- `SSH_PORT` — defaults to `22`
- `DEPLOY_PATH` — defaults to `/opt/lison`
- `SERVER_GOARCH` — defaults to `amd64`; set `arm64` for an ARM server

## One-time server setup

1. Install Docker Engine and the Compose plugin.
2. Create `/opt/lison` and copy `compose.production.yaml` there as `/opt/lison/compose.production.yaml`.
3. Copy `production.env.example` to `/opt/lison/.env`, fill the values and run `chmod 600 /opt/lison/.env`.
   Use a URL-safe PostgreSQL password because it is inserted into `DATABASE_URL` by Compose.
4. Give the deployment user access to Docker and `/opt/lison`.
5. Log in to GHCR once on the server. Use a GitHub token with `read:packages` when the packages are private.
6. Start PostgreSQL once:

   ```bash
   cd /opt/lison
   docker compose -f compose.production.yaml up -d postgres
   ```

After this, pushes to `main` deploy each container independently. The deployment performs a local health-check and restores the previous image if it fails.

Nginx can later proxy `/` to `http://127.0.0.1:3000` and `/api/` plus `/health` to `http://127.0.0.1:8080`.
