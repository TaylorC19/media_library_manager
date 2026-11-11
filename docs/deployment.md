# Deployment

## Overview

The Go application is built from the root [`Dockerfile`](../Dockerfile) (Alpine-based, static binary `web`). Compose files:

- [`docker-compose.yml`](../docker-compose.yml) — **local development only:** MongoDB (run the Go binary on the host with `./scripts/dev-web.sh`)
- [`docker-compose.prod.yml`](../docker-compose.prod.yml) — **production:** Go **app** container; optional **mongo** service via profile

Deployment stays intentionally simple: no Kubernetes in-repo, no bundled TLS reverse proxy.

MongoDB connection uses **`MONGODB_URI`** or **`MONGODB_URL`** (alias) and **`MONGODB_DATABASE`**, as read by [`internal/config/config.go`](../internal/config/config.go).

## Image and runtime

- **Build context:** repository root
- **Binary:** runs as non-root user `app` from `/home/app/web`, listens on `PORT` (default `8080`)
- **TLS to MongoDB and providers:** the runtime image installs **`ca-certificates`** (Alpine) so `mongodb+srv://` (Atlas) and HTTPS provider APIs trust public CAs.
- **Templates/static:** when `APP_ENV=production`, the app uses embedded assets (`internal/config.UseEmbeddedAssets`). The production Compose service sets `APP_ENV=production` so the container works without mounting source.

When **`APP_ENV=development`** (typical local `./scripts/dev-web.sh`), templates and static files are read from the working tree under `internal/views` and `internal/static` so edits apply on restart (or live with `air`).

When **`APP_ENV=production`**, templates, locale JSON, and `internal/static/public/**` are served from `go:embed` data baked into the binary—no bind-mount of source is required in Docker.

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

- **`MONGODB_URI`** — Atlas (`mongodb+srv://...`) or another replica set
- **`PUBLIC_ORIGIN`** — the exact HTTPS origin
- **`PROVIDER_KEYS_ENCRYPTION_KEY`** — base64 encoding of 32 random bytes
- **`SESSION_COOKIE_NAME`** — a `__Host-` name
- **`SESSION_COOKIE_SECURE`** — must be `true`
- Provider keys and **`MUSICBRAINZ_USER_AGENT`** as needed

There is **no** `SESSION_SECRET` in the Go app: sessions use stored tokens in MongoDB (see auth service). Use strong cookie settings and HTTPS in production.

### 2. Preview the resolved config

```bash
make docker-prod-config
```

Resolved configuration can contain credentials. Inspect it locally; do not commit it or paste it into public logs, issues, or QA reports.

### 3. Start

External or Atlas Mongo:

```bash
make docker-prod-up
```

The optional bundled Mongo profile needs replica-set initialization and index
setup before starting the app. On a fresh installation:

```sh
docker compose --env-file .env.prod -f docker-compose.prod.yml --profile mongo up -d mongo
docker compose --env-file .env.prod -f docker-compose.prod.yml exec mongo mongosh --quiet --eval 'rs.initiate({_id:"mlm",members:[{_id:0,host:"mongo:27017"}]})'
```

Wait for the primary, then run `cmd/dbsetup` with maintenance credentials from a
host/container that can resolve the replica-set member address. Set
`MONGODB_URI=mongodb://mongo:27017/?replicaSet=mlm` for the app and start it with
`make docker-prod-up-mongo`. Initialize once per new database volume; configure
MongoDB authentication/network isolation for your hosting environment.

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

Set `MONGODB_URI` to your replica-set URI. Production does not accept a standalone database.

## Health checks

- **`GET /health`** — JSON `{ "ok": true, ... }`. Docker health checks use `wget` against `http://127.0.0.1:8080/health` (default `PORT`).

## HTTPS and cookies

Set `SESSION_COOKIE_SECURE=true` when the app is served behind HTTPS so session cookies are marked `Secure`. Terminate TLS at your reverse proxy or platform edge.

## Provider environment

Production templates include provider keys and cache TTLs (`PROVIDER_CACHE_*`, MusicBrainz throttling). See [`.env.prod.example`](../.env.prod.example) and [`internal/config/config.go`](../internal/config/config.go).

## Database backup and restore

Full-database backup options are documented in [`docs/backup.md`](backup.md).

**Quick ops backup** (requires MongoDB Database Tools on the host):

```bash
make db-backup
make db-restore ARCHIVE=backups/mongodump-<timestamp>
```

With bundled Mongo in Docker:

```bash
docker compose exec mongo mongodump --uri=mongodb://127.0.0.1:27017 --db=media_library --out=/tmp/backup
```

The production app container does not include `mongodump`; use host tools, the `mongo` service, or `go run ./cmd/db export` during controlled maintenance.

## Raspberry Pi notes

- Prefer **64-bit** OS.
- The Go binary and `mongo:7` images support **arm64**.
- Atlas often reduces load on the Pi versus running Mongo locally.
- Building on the Pi is straightforward; use `buildx` if you need multi-arch publishing.


## Vercel

The root `vercel.json` selects the Go preset and builds `cmd/server` as the server.
Vercel supplies `PORT`; templates and assets are embedded. The MongoDB client
is reused for the lifetime of each server with a bounded connection pool.
See [Vercel Go runtime documentation](https://vercel.com/docs/functions/runtimes/go).

Set `APP_ENV=production`, `PUBLIC_ORIGIN` to the exact HTTPS origin (no path),
`SESSION_COOKIE_SECURE=true`, `SESSION_COOKIE_NAME=__Host-mlm_session`,
`MONGODB_URI`, `MONGODB_DATABASE`, and `PROVIDER_KEYS_ENCRYPTION_KEY` in Vercel
project secrets. Generate the last value with `openssl rand -base64 32`; keep
it separate from database backups and retain it for credential recovery.
Changing it requires re-encrypting existing personal keys or re-entering them.
Use independent databases, encryption keys and provider credentials for Preview
and Production, and restrict preview access with deployment protection.
For previews, set the exact preview origin; an incorrect origin rejects writes.
Real environment files are excluded from Git and deployment uploads.

Production requires a MongoDB replica set (Atlas is suitable). Before deployment,
run `go run ./cmd/dbsetup` from a trusted workstation with maintenance database
credentials and the intended production configuration. It creates indexes, TTL
cleanup and the admin transaction guard. Keep `ENSURE_INDEXES=false` in runtime
settings; grant runtime credentials only the collection read/write permissions
needed by the app. Restrict database network access to the deployment's egress
or private networking; verify your Vercel plan's networking configuration.
Create the first administrator from a trusted terminal with:

```sh
go run ./cmd/users -create -username admin -role admin -password-stdin < /path/to/private/password-file
```

Use an owner-only temporary password file or pipe from a password manager; remove
the temporary file afterward. The password is never a command-line argument.
The existing `cmd/users -username ... -role ...` command remains the recovery path.
`SIGNUP_ENABLED=false` is the production default; admins create subsequent users.
Public registration is an explicit opt-in and remains subject to shared limits.

Personal provider overrides are encrypted in `personal_provider_keys` in MongoDB,
using AES-256-GCM with owner/provider binding. They never reach browser forms or
the custom app backup archive. Development can still use the local key directory;
existing local keys are not automatically migrated. Enter desired keys in the
new deployment's Settings. No persistent application writes use Vercel's filesystem.

Web database export/restore is disabled in production. Use managed database
snapshots or database tools during controlled maintenance, restoring into a new
isolated database and validating it before switching the runtime connection.
Include the encrypted credential collection in managed snapshots; store its
encryption key separately. Test this recovery process in staging.

Production uses shared MongoDB request budgets: authentication limits per IP and
account, provider limits of 60 requests/minute and 1,000/day per user, at most eight
concurrent provider requests per user, and 500 writes/day. Database failures reject
budgeted requests. Vercel's trusted client-IP header is used only on Vercel;
standalone servers use the direct peer address. Configure edge rate limits as an
additional traffic control. Boundaries apply to requests, so an All media search
uses several provider requests. Budget windows reset at fixed UTC boundaries.

Before switching a production domain, verify HTTPS login/logout, account roles,
origin rejection, provider settings across restarts, search dialogs, camera
scanning, and logout/back navigation on a protected staging deployment. Run the
release workflow; local tests do not verify cloud networking or project secrets.
