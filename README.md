# Media Library Manager Alpha Version 0.1.0

A self-hosted media library app for tracking movies, TV, albums, books, and games in one private collection.

`Media Library Manager` is built for people who want to keep their own catalog instead of spreading it across spreadsheets, notes, and closed third-party services. The app is a **single Go web application** (server-rendered HTML) with MongoDB persistence and server-side metadata providers.

In v1, every item lives in one of two buckets:

- `catalog` for things you already own
- `wishlist` for things you want to track without mixing them into the main collection

## Why Use It

- Keep your media library in a self-hosted app you control
- Track multiple media types in one place
- Separate your personal notes and ownership data from imported metadata
- Use provider-backed search instead of entering every item by hand
- Use barcode scanning as a fast lookup tool for physical media
- Run it locally with Docker or deploy it as a small private web app

## Features

- Private, login-first application
- Username/password auth with HTTP-only session cookies
- Catalog and wishlist workflows
- Movies, TV, albums, books, and games
- Manual entry creation when provider data is missing or incomplete
- Provider-backed search with normalized results
- Barcode-assisted lookup with confirmation-first matching
- English and Japanese UI support
- Docker Compose for local **MongoDB** only; production Go image via `docker-compose.prod.yml`
- `GET /health` for readiness and container health checks

## Philosophy

- the browser never talks to third-party metadata providers directly
- the Go server owns auth, provider access, normalization, caching, and barcode lookup
- user-owned collection data is stored separately from imported metadata
- barcode scanning is treated as candidate matching, not silent automation

## Why Go

A single Go binary keeps deployment small and predictable: one process to run on a home server or a Raspberry Pi, explicit layering (handlers → services → repositories), and strong control over concurrency and resource use. Provider HTTP calls, normalization, caching, and sessions stay in one codebase—no separate “frontend” and “API” deployables to version together.

## Why HTML-first (and not a SPA)

Pages are rendered on the server with `html/template`. That makes auth, links, and forms straightforward, works well for a private catalog app, and keeps **canonical state on the server**. The UI is **not** a single-page React or Vue app; it also is **not** “no JavaScript.” **[htmx](https://htmx.org/)** swaps in HTML fragments for search results, list filters, pagination, and inline notices so interactions feel responsive without moving business logic into the browser. **Vanilla JS** is used where the platform requires it—especially the **scan** flow (camera and barcode decoding). See [`docs/templates.md`](docs/templates.md).

## Quick Start (recommended)

1. **MongoDB in Docker** (database only — fast iteration on the Go app on your host):

```bash
./scripts/dev-db-up.sh
```

2. **Run the Go app** with live reload:

```bash
./scripts/dev-web.sh
```

3. Open [http://localhost:8080](http://localhost:8080)

`./scripts/dev-web.sh` uses `air` when available so template, locale, and static asset changes restart the server. For a one-shot run without `air`:

```bash
./scripts/dev-web-run.sh
```

4. Copy [`.env.example`](.env.example) to `.env` and adjust. For local Mongo, `MONGODB_URI=mongodb://localhost:27017` matches `./scripts/dev-db-up.sh`.

## Docker Compose (local dev)

[`docker-compose.yml`](docker-compose.yml) runs **only MongoDB** for development. Start the DB, then run the Go app on your machine (see Quick Start above):

```bash
make docker-up
# or: docker compose up -d
```

Use `MONGODB_URI=mongodb://localhost:27017` in `.env` when the app runs on the host.

For **production**, build and run the Go app in Docker with [`docker-compose.prod.yml`](docker-compose.prod.yml) and the root [`Dockerfile`](Dockerfile) — see [Deployment](#deployment) and [`docs/deployment.md`](docs/deployment.md).

## Configuration

Key variables are documented in [`.env.example`](.env.example) and loaded in [`internal/config/config.go`](internal/config/config.go). They include MongoDB, session cookie settings, provider API keys, MusicBrainz throttling, and provider cache TTLs.

Templates:

- `.env.example` — local development
- `.env.prod.example` — copy to `.env.prod` for production-style Compose (see [`docs/deployment.md`](docs/deployment.md))

## Providers

- **TMDB** — movies and TV
- **MusicBrainz** — albums
- **Discogs** — music enrichment and some barcode support
- **Open Library** — books
- **RAWG** — games

Details: [`docs/provider-notes.md`](docs/provider-notes.md).

## Deployment

Production-style Compose (see [`.env.prod.example`](.env.prod.example)):

```bash
cp .env.prod.example .env.prod
# edit .env.prod (especially MONGODB_URI)
make docker-prod-up
```

Optional bundled Mongo:

```bash
make docker-prod-up-mongo
```

Full notes: [`docs/deployment.md`](docs/deployment.md).

## Health checks

- `GET /health` — JSON status; used by Docker health checks.

## Architecture

The codebase is a **Go monolith**:

- `cmd/web` — entrypoint
- `internal/http` — routes, middleware, handlers
- `internal/views` — `html/template` and locales
- `internal/service` — business logic
- `internal/repository` — MongoDB access
- `internal/providers` — TMDB, MusicBrainz, etc.

Further reading:

- [`docs/spec.md`](docs/spec.md) — product rules and technical constraints
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/routes.md`](docs/routes.md) and [`docs/api.md`](docs/api.md)
- [`docs/data-model.md`](docs/data-model.md)
- [`docs/migration-from-legacy.md`](docs/migration-from-legacy.md) — retired Node monorepo (historical)

## Project layout

```txt
cmd/web/           main
internal/          app code (HTTP, services, repos, providers, views, static embed)
docs/              spec, architecture, deployment, routes, templates, data model
Dockerfile         multi-stage Go build
docker-compose.yml mongo only (local dev DB); docker-compose.prod.yml Go app + optional mongo
```

## Roadmap

- richer settings and account management
- import and export flows
- saved filters and collection views
- deeper edition and release modeling for physical media
- background metadata refresh
- replace [`docs/screenshots/README.md`](docs/screenshots/README.md) placeholders with real captures; more deployment examples
