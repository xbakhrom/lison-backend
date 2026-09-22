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
2. Create `/opt/lison`, copy `compose.production.yaml` there as
   `/opt/lison/compose.production.yaml`, and copy the `nginx` directory there
   as `/opt/lison/nginx`.
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

## Nginx and HTTPS

The checked-in Nginx configuration serves `lison.xbakhrom.uz`, sends `/api/`
and `/health` to the backend on `127.0.0.1:8080`, and sends every other
request to the webapp on `127.0.0.1:3010`.

Before starting, point the domain's DNS record to the server and allow inbound
TCP ports 80 and 443. On Ubuntu/Debian, install Nginx and Certbot:

```bash
sudo apt update
sudo apt install -y nginx certbot
sudo install -d -m 755 /var/www/certbot
```

Install the HTTP-only bootstrap configuration first:

```bash
cd /opt/lison
sudo install -m 644 nginx/lison.bootstrap.conf /etc/nginx/sites-available/lison
sudo ln -sfn /etc/nginx/sites-available/lison /etc/nginx/sites-enabled/lison
sudo nginx -t
sudo systemctl reload nginx
```

Request the first certificate, then replace the bootstrap configuration with
the production HTTPS configuration:

```bash
sudo certbot certonly --webroot -w /var/www/certbot \
  --deploy-hook "systemctl reload nginx" \
  -d lison.xbakhrom.uz
sudo install -m 644 nginx/lison.conf /etc/nginx/sites-available/lison
sudo nginx -t
sudo systemctl reload nginx
```

Verify the public routes and automatic renewal:

```bash
curl -I http://lison.xbakhrom.uz
curl https://lison.xbakhrom.uz/health
sudo certbot renew --dry-run
```

The HTTP request should redirect to HTTPS, and the health endpoint should
return `{"status":"ok"}`. Certbot's systemd timer will renew the certificate;
its webroot challenge continues to work through the production configuration.
