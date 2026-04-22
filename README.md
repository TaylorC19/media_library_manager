# Media Library Manager Version 1.0.0

A self-hosted media library app for tracking movies, TV, albums, books, and games in one private collection.

`Media Library Manager` is built for people who want to keep their own catalog instead of spreading it across spreadsheets, notes, and closed third-party services. It combines a clean web UI with a backend that handles auth, metadata lookups, normalization, and barcode-assisted workflows.

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
- Docker Compose for development and production-style deployment
- `GET /health` endpoint for readiness and container health checks

## Philosophy

This project is meant to feel like a practical self-hosted app:

- the frontend never talks to third-party metadata providers directly
- the API owns auth, provider access, normalization, caching, and barcode lookup
- user-owned collection data is stored separately from imported metadata
- barcode scanning is treated as candidate matching, not silent automation

That makes the app easier to reason about, easier to extend, and safer to run as a private personal service.

## Quick Start

### Go App (fast local dev)

For the current root-level Go app, the fastest feedback loop is:

1. Start Mongo only in Docker:

```bash
./scripts/dev-db-up.sh
```

2. Run the Go web app on your host machine with live reload:

```bash
./scripts/dev-web.sh
```

3. Open [http://localhost:8080](http://localhost:8080)

`./scripts/dev-web.sh` runs the Go app through `air`, so backend, template, locale, and static asset changes restart the server automatically. If you want the old one-shot behavior, use:

```bash
./scripts/dev-web-run.sh
```

This still avoids rebuilding the web container on every code change.

### Docker

The easiest way to run the app locally is with Docker Compose.

1. Create a local env file: 

```bash
cp .env.example .env
```

2. Start the development stack:

```bash
pnpm docker:dev:up
```

3. Open:

- web: [http://localhost:3000](http://localhost:3000)
- API health: [http://localhost:4000/health](http://localhost:4000/health)

This starts:

- `web`
- `api`
- `mongo`

The Compose file wires container-to-container networking automatically, so the web app talks to `api` and the API talks to `mongo`.

### Without Docker

If you want to run the services directly on your machine:

```bash
cp .env.example .env
corepack enable
pnpm install
pnpm dev
```

You will need a running MongoDB instance at the `MONGODB_URL` from `.env`.

## Configuration

Important env vars:

- `NEXT_PUBLIC_API_BASE_URL`
- `INTERNAL_API_BASE_URL`
- `API_PORT`
- `WEB_PORT`
- `MONGODB_URL`
- `SESSION_COOKIE_NAME`
- `SESSION_SECRET`
- `CORS_ORIGIN`

Provider integrations also use env vars for API keys, base URLs, and cache behavior.

Templates:

- `.env.example` for local development
- `.env.prod.example` for production-style deployment

## Providers

The current provider setup is:

- `TMDB` for movies and TV
- `MusicBrainz` for albums
- `Discogs` for music enrichment and some barcode support
- `Open Library` for books
- `RAWG` for games

These are treated as backend adapters, not frontend dependencies. The app stores normalized records and provider references rather than mirroring raw external payloads.

More notes are in [`docs/provider-notes.md`](docs/provider-notes.md).

## Screenshots

Suggested screenshots for this project:

- login screen
- dashboard
- catalog or wishlist view
- provider search results
- barcode scan flow
- library detail page

If you want to add screenshots to the repo, place them in `docs/screenshots/` and link them here.

## Deployment

For a production-style deployment:

```bash
cp .env.prod.example .env.prod
pnpm docker:prod:up
```

If you want to run the bundled Mongo container instead of Atlas or another external Mongo instance:

```bash
pnpm docker:prod:up:mongo
```

Notes:

- the web image uses Next.js standalone output
- the API image runs compiled NestJS output
- the API health route is used by Docker health checks
- auth cookies are `Secure` in production, so browser access should be behind HTTPS

See [`docs/deployment.md`](docs/deployment.md) for the full deployment guide, Mongo switching options, and Raspberry Pi notes.

## Health Checks

The API exposes:

- `GET /health`

It returns API status and Mongo connection state and is used by the Compose health checks.

## Architecture

This repo uses a split web/API architecture:

- `apps/web` is the Next.js frontend
- `apps/api` is the NestJS backend
- `packages/types` contains shared contracts
- `packages/provider-sdk` contains provider-facing contracts and helpers

That structure keeps provider access, normalization, auth, and persistence logic on the server where they belong.

If you want the deeper system breakdown, see:

- [`docs/architecture.md`](docs/architecture.md)
- [`docs/api.md`](docs/api.md)
- [`docs/data-model.md`](docs/data-model.md)

## Project Structure

```txt
apps/
  api/            NestJS API
  web/            Next.js frontend
packages/
  config/         Shared config helpers
  provider-sdk/   Provider contracts and normalization helpers
  types/          Shared domain and API types
docs/
  architecture.md
  api.md
  data-model.md
  deployment.md
  provider-notes.md
```

## Roadmap

- richer settings and account management
- import and export flows
- saved filters and collection views
- deeper edition and release modeling for physical media
- background metadata refresh
- screenshot assets and more deployment examples
