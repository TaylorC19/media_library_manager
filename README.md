# Media Library Manager

A self-hostable full-stack TypeScript app for managing personal collections across movies, TV, music, books, and games.

## Why I Built It

This project is meant to showcase:

- full-stack TypeScript architecture
- a clean Next.js and NestJS separation
- MongoDB-backed persistence
- provider-based metadata ingestion
- normalization and shared type contracts
- Docker-friendly local development

## Current Status

The repo is currently at the initial monorepo bootstrap stage.

Implemented so far:

- `pnpm` workspace + Turborepo
- `apps/web` with Next.js, TypeScript, and Tailwind
- `apps/api` with NestJS, Mongo wiring, and `GET /health`
- `packages/types` for shared domain types
- `packages/provider-sdk` for provider contracts and mapping helpers
- `packages/config` for shared TypeScript and env utilities
- `docker-compose.yml` for `web`, `api`, and `mongo`
- root `.env.example`

## Architecture

- Frontend: Next.js (`apps/web`)
- Backend: NestJS (`apps/api`)
- Database: MongoDB
- Shared domain contracts: `packages/types`
- Provider abstraction layer: `packages/provider-sdk`
- Shared config/helpers: `packages/config`

The frontend must not call third-party metadata providers directly. All provider work, auth, sessions, normalization, and barcode lookup orchestration belong in the API.

## Repo Layout

```txt
apps/
  web/            Next.js frontend
  api/            NestJS backend
packages/
  types/          Shared domain and API types
  provider-sdk/   Provider contracts and normalization helpers
  config/         Shared TypeScript config and env helpers
docs/
  product-spec.md
```

## Getting Started

### 1. Create your env file

```bash
cp .env.example .env
```

You can keep the default values for local development.

### 2. Start with Docker Compose

This is the easiest way to run the current scaffold because the containers install dependencies and start the web app, API, and MongoDB together.

```bash
docker compose up
```

Once the services are up:

- Web: [http://localhost:3000](http://localhost:3000)
- API health: [http://localhost:4000/health](http://localhost:4000/health)
- MongoDB: `mongodb://localhost:27017`

Docker Compose overrides the API container to use `mongo` internally, so the root `.env` can stay host-friendly with `localhost`.

### 3. Run locally with pnpm instead

If you already have Node 22+ and `pnpm` available:

```bash
corepack enable
pnpm install
pnpm dev
```

Useful scripts:

- `pnpm dev`
- `pnpm dev:web`
- `pnpm dev:api`
- `pnpm build`
- `pnpm lint`
- `pnpm typecheck`

## Environment Variables

The root `.env.example` includes:

- app runtime: `NODE_ENV`
- web config: `WEB_PORT`, `NEXT_PUBLIC_API_BASE_URL`
- api config: `API_PORT`, `APP_BASE_URL`, `SESSION_COOKIE_NAME`, `SESSION_SECRET`, `CORS_ORIGIN`
- mongo config: `MONGODB_URL`
- provider config: TMDB, MusicBrainz, Discogs, Open Library, and RAWG settings

Provider keys are included now so future milestones can add integrations without reshaping the env contract.

Use `mongodb://localhost:27017/media_library` when connecting from your Mac or Mongo Compass. The Docker-only hostname `mongo` is only valid from other Compose containers.

## Verification

The current scaffold has been verified with:

```bash
pnpm build
pnpm typecheck
```

## Roadmap

Next milestones from the product spec:

- authentication with session cookies
- protected app routes
- library and media collections
- provider-backed search and import
- barcode-assisted lookup
- portfolio-ready docs and deployment polish
