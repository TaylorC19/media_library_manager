# Deployment

## Overview

This repo ships with two Docker Compose flows:

- `docker-compose.yml` for local development
- `docker-compose.prod.yml` for production-style self-hosting

The deployment story stays intentionally simple:

- no Kubernetes
- no separate infra repo
- no bundled TLS or reverse proxy
- one Mongo connection variable: `MONGODB_URL`

## Images And Runtime

### Web

- built from `apps/web/Dockerfile`
- uses Next.js standalone output
- starts with `node apps/web/server.js`

### API

- built from `apps/api/Dockerfile`
- compiles NestJS into `dist`
- starts with `node dist/main.js`

Both Dockerfiles use `node:22-alpine`.

## Local Development

### 1. Create `.env`

```bash 
cp .env.example .env
```

The root `.env` is host-friendly by default:

- `NEXT_PUBLIC_API_BASE_URL=http://localhost:4000`
- `INTERNAL_API_BASE_URL=http://localhost:4000`
- `MONGODB_URL=mongodb://localhost:27017/media_library`

The dev Compose file overrides the container-only values that should resolve to Docker service names instead.

### 2. Start the dev stack

```bash
pnpm docker:dev:up
```

Services:

- `deps`: installs workspace dependencies into the shared volume
- `mongo`: local MongoDB on `mongodb://localhost:27017`
- `api`: NestJS watch-mode server on `http://localhost:4000`
- `web`: Next.js dev server on `http://localhost:3000`

### 3. Stop or reset

```bash
pnpm docker:dev:down
pnpm docker:dev:reset
```

`docker:dev:reset` also removes volumes.

## Production-Style Deployment

### 1. Create `.env.prod`

```bash
cp .env.prod.example .env.prod
```

Required edits:

- set `SESSION_SECRET` to a strong random string
- set `APP_BASE_URL` to the real web origin
- set `CORS_ORIGIN` to the same browser-facing web origin
- set `NEXT_PUBLIC_API_BASE_URL` to the browser-facing API origin
- set `MONGODB_URL` to Atlas, bundled Mongo, or another external Mongo instance
- keep `INTERNAL_API_BASE_URL=http://api:4000` when `web` and `api` stay on the same Compose network

### 2. Preview the resolved Compose config

```bash
pnpm docker:prod:config
```

### 3. Start the stack

Atlas or another external Mongo:

```bash
pnpm docker:prod:up
```

Bundled self-hosted Mongo:

```bash
pnpm docker:prod:up:mongo
```

### 4. Operate the stack

```bash
pnpm docker:prod:logs
pnpm docker:prod:down
```

## MongoDB Switching

### Local Docker dev

Keep the host-friendly root value:

```env
MONGODB_URL=mongodb://localhost:27017/media_library
```

The Compose file overrides the API container to:

```env
MONGODB_URL=mongodb://mongo:27017/media_library
```

### Production with MongoDB Atlas

Set:

```env
MONGODB_URL=mongodb+srv://<user>:<password>@<cluster>/<db>?retryWrites=true&w=majority
```

Then run:

```bash
pnpm docker:prod:up
```

### Production with bundled self-hosted Mongo

Set:

```env
MONGODB_URL=mongodb://mongo:27017/media_library
```

Then run:

```bash
pnpm docker:prod:up:mongo
```

The `mongo` service is behind an optional Compose profile, so Atlas users do not need to run a database container.

### Production with another external Mongo instance

Set:

```env
MONGODB_URL=mongodb://<host>:27017/media_library
```

Then run:

```bash
pnpm docker:prod:up
```

## Health Checks And Startup Order

### Development

- `deps` must complete before the app services start
- `api` waits for a healthy `mongo`
- `web` waits for a healthy `api`

### Production

- `api` exposes `GET /health`
- `web` waits for a healthy `api`
- `mongo` has its own health check when the `mongo` profile is enabled
- `api` marks `mongo` as an optional dependency so Atlas and external Mongo stay valid

The API health route returns service status plus Mongo readiness and is the correct first endpoint to test after boot.

## HTTPS And Cookies

When `NODE_ENV=production`, the API marks auth cookies as `Secure`. Browser-based auth should therefore run behind HTTPS.

This repo does not bundle TLS termination. For an internet-facing deployment, place the stack behind a trusted HTTPS-capable reverse proxy or platform edge.

## Provider Env Notes

Production env templates include provider-specific settings for:

- TMDB
- MusicBrainz
- Discogs
- Open Library
- RAWG
- provider cache TTLs and cache versioning

MusicBrainz-specific throttling env vars are especially important for stability in self-hosted deployments.

## Raspberry Pi Notes

- use a 64-bit OS on the Pi
- `node:22-alpine` works well for both amd64 and arm64 builds
- `mongo:7` is suitable for the optional bundled database on 64-bit ARM
- Atlas is usually the easiest path on a Pi because it removes local database CPU, RAM, and disk pressure
- if you self-host Mongo on the Pi, use reliable storage and expect slower image builds than on desktop hardware
- build directly on the Pi for the simplest flow, or adopt `docker buildx` later if you want multi-arch image publishing
