# Deployment

## Overview

The Go application is built from the root [`Dockerfile`](../Dockerfile) (Alpine-based, static binary `web`). Compose files:

- [`docker-compose.yml`](../docker-compose.yml) — **local development only:** MongoDB (run the Go binary on the host with `./scripts/dev-web.sh`)
- [`docker-compose.prod.yml`](../docker-compose.prod.yml) — **production:** Go **app** container; optional **mongo** service via profile

Deployment stays intentionally simple: no Kubernetes in-repo, no bundled TLS reverse proxy.

MongoDB connection uses **`MONGODB_URI`** or **`MONGODB_URL`** (alias) and **`MONGODB_DATABASE`**, as read by [`internal/config/config.go`](../internal/config/config.go).

## Image and runtime

- **Build context:** repository root
- **Binary:** `/home/app/web` in the runtime image, listens on `PORT` (default `8080`)
- **Templates/static:** when `APP_ENV=production`, the app uses embedded assets (`internal/config.UseEmbeddedAssets`). The production Compose service sets `APP_ENV=production` so the container works without mounting source.

## Local development (Mongo in Docker, Go on the host)

### 1. Create `.env`

```bash
cp .env.example .env
```

The Go process on your machine should use **localhost** to reach the published Mongo port:

```env
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=media_library
```

### 2. Start MongoDB

```bash
make docker-up
# or: docker compose up -d
# or: ./scripts/dev-db-up.sh
```

### 3. Run the Go app

```bash
./scripts/dev-web.sh
```

### 4. Stop Mongo

```bash
make docker-down
# or: docker compose down
```

To remove volumes: `docker compose down -v`.

## Production-style Compose

### 1. Create `.env.prod`

```bash
cp .env.prod.example .env.prod
```

Edit at minimum:

- **`MONGODB_URI`** — Atlas (`mongodb+srv://...`), bundled `mongo` service (`mongodb://mongo:27017`), or another host
- **`SESSION_COOKIE_SECURE`** — `true` when serving over HTTPS
- Provider keys and **`MUSICBRAINZ_USER_AGENT`** as needed

There is **no** `SESSION_SECRET` in the Go app: sessions use stored tokens in MongoDB (see auth service). Use strong cookie settings and HTTPS in production.

### 2. Preview the resolved config

```bash
make docker-prod-config
```

### 3. Start

External or Atlas Mongo:

```bash
make docker-prod-up
```

With bundled Mongo:

```bash
make docker-prod-up-mongo
```

Ensure `.env.prod` uses `MONGODB_URI=mongodb://mongo:27017` when using the `mongo` profile.

### 4. Logs and stop

```bash
make docker-prod-logs
make docker-prod-down
```

## MongoDB switching

### Compose `mongo` service

Use host/port `mongo:27017` from the app container.

### MongoDB Atlas

Set `MONGODB_URI` to your `mongodb+srv://` connection string. Do not start the `mongo` profile.

### Self-hosted Mongo elsewhere

Set `MONGODB_URI` to your replica set or standalone URI.

## Health checks

- **`GET /health`** — JSON `{ "ok": true, ... }`. Docker health checks use `wget` against `http://127.0.0.1:8080/health` (default `PORT`).

## HTTPS and cookies

Set `SESSION_COOKIE_SECURE=true` when the app is served behind HTTPS so session cookies are marked `Secure`. Terminate TLS at your reverse proxy or platform edge.

## Provider environment

Production templates include provider keys and cache TTLs (`PROVIDER_CACHE_*`, MusicBrainz throttling). See [`.env.prod.example`](../.env.prod.example) and [`internal/config/config.go`](../internal/config/config.go).

## Raspberry Pi notes

- Prefer **64-bit** OS.
- The Go binary and `mongo:7` images support **arm64**.
- Atlas often reduces load on the Pi versus running Mongo locally.
- Building on the Pi is straightforward; use `buildx` if you need multi-arch publishing.
