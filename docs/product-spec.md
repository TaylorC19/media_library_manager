Below is the full build package.

I’ve locked the external-provider assumptions to current official docs: MongoDB Atlas still has a free tier with 512 MB storage, TMDB is free for non-commercial use with attribution, MusicBrainz enforces a 1 request/second client limit, Open Library provides public-good APIs for book lookup/discovery, Discogs splits API data into CC0 vs restricted data, and RAWG allows free personal use with attribution. ([MongoDB][1])

---

# 1) Project definition

## Working concept

A **self-hostable personal media library manager** for:

* movies
* TV shows
* music albums
* books
* games

Core behavior:

* private, login-first app
* each user has two buckets: **catalog** and **wishlist**
* barcode scanning is **best-effort candidate matching**
* ratings are **read-only external metadata** if a provider exposes them
* no user review/rating system in v1
* built for portfolio value with strong backend architecture, clean docs, and real integrations

## Product goals

The project should demonstrate:

* full-stack architecture
* TypeScript across frontend and backend
* monorepo organization
* authentication and session handling
* external API provider abstraction
* metadata normalization
* caching and rate-limit awareness
* mobile-web barcode scanning
* Dockerized development and production flows
* self-hosting readiness on a Raspberry Pi

## Non-goals for v1

Do not build these now:

* social feed
* public profiles
* password reset or email verification
* native mobile app
* file/media playback
* loan tracking
* multi-library-per-user
* user-created ratings/reviews
* advanced edition hierarchy beyond what is necessary for imports

---

# 2) Recommended stack

## Frontend

* **Next.js**
* **TypeScript**
* **Tailwind CSS**
* **TanStack Query**
* **Zod**
* **React Hook Form**
* mobile-first responsive UI
* PWA-ready structure, but PWA can be deferred

## Backend

* **NestJS**
* **TypeScript**
* REST API
* session-based auth
* provider abstraction layer
* caching layer
* validation via Zod or class-validator, but keep request/response types clear

## Database

* **MongoDB**

  * local Docker MongoDB for dev
  * MongoDB Atlas for optional hosted prod connection

Atlas free tier is still suitable for a small personal/portfolio deployment. ([MongoDB][1])

## Auth

* local username/password
* secure password hashing
* session cookie auth
* no email flows
* protected routes on both API and web

## Barcode scanning

* browser/mobile-web camera scanning
* use a proven JS barcode library
* scan UPC/EAN/ISBN style codes
* no QR support required in v1

## Deployment

* monorepo
* Docker Compose for dev
* separate “prod” compose or override for Raspberry Pi deployment
* backend and frontend containerized
* Mongo container in dev
* prod can point either to Atlas or to a self-hosted Mongo later

---

# 3) External provider strategy

## Movies / TV

Use **TMDB** as the primary source.

Reason:

* simple API
* strong movie/TV search coverage
* images
* external IDs
* non-commercial use allowed with attribution

TMDB’s official FAQ says the API is free for non-commercial purposes with attribution. ([The Movie Database (TMDB)][2])

## Music

Use:

* **MusicBrainz** as the primary metadata source
* **Discogs** as optional enrichment for release-specific physical media details

Reason:

* MusicBrainz is very suitable for structured music metadata, but enforce rate limiting carefully
* Discogs is useful for physical-release metadata like barcodes and release details, but some API data is CC0 while images are restricted, so do not architect around mirroring Discogs images/data blindly

MusicBrainz requires each client app to stay at or below 1 call per second and to send a proper User-Agent. Discogs documents rate-limiting behavior and distinguishes CC0 data from restricted data, including release images. ([MusicBrainz][3])

## Books

Use **Open Library**.

Reason:

* open/public-good orientation
* ISBN lookup support
* cover APIs
* easy fit for barcode/identifier lookups

Open Library’s developer docs explicitly position the APIs for public-good book lookup/discovery, and the legacy Books API supports ISBN and other identifiers. ([Open Library][4])

## Games

Use a provider abstraction from day one. For v1, default to **RAWG**.

Reason:

* simpler than IGDB
* personal-use friendly
* easier to start than Twitch-based auth flows

Caveat:

* games metadata is the weakest/most replaceable provider area in this system
* keep it behind an adapter interface so you can replace RAWG later with IGDB or another source

RAWG’s docs say the API is free for personal use with attribution. IGDB is another viable option, but its official getting-started flow requires Twitch developer registration and OAuth. ([api.rawg.io][5])

## Important storage rule

Do **not** design the app as a full mirror of third-party metadata.
Instead:

* keep a normalized internal media record
* keep provider IDs
* store a **limited metadata snapshot**
* refresh from source when needed
* respect provider attribution and restricted-image constraints where applicable

That is especially important for TMDB and Discogs. ([The Movie Database (TMDB)][2])

---

# 4) Architecture

## High-level shape

### apps

* `apps/web` → Next.js frontend
* `apps/api` → NestJS backend

### packages

* `packages/types` → shared TypeScript domain types
* `packages/config` → shared lint/tsconfig/env helpers
* `packages/provider-sdk` → provider contracts + mappers
* `packages/ui` → optional shared UI components later

## Architectural rules

1. The frontend never talks directly to third-party metadata APIs.
2. All provider calls go through the backend.
3. The backend owns:

   * auth
   * sessions
   * permissions
   * provider integration
   * caching
   * normalization
   * barcode lookup orchestration
4. Shared domain types live in `packages/types`.
5. The API should be designed so that a mobile app could consume it later without major redesign.

## Suggested domain boundaries

* **auth**
* **users**
* **library**
* **media**
* **providers**
* **barcode**
* **search**
* **health/system**

---

# 5) Data model

Do **not** use SQL-style subtype tables like `item_as_cd` or `item_as_game`.

Use:

* `users`
* `sessions`
* `library_entries`
* `media_records`
* optional `provider_cache`
* optional `scan_logs`

## Core idea

Separate:

### A) user-owned app data

This belongs in `library_entries`

### B) external metadata

This belongs in `media_records`

---

## `users`

```ts
interface User {
  _id: string;
  username: string;
  passwordHash: string;
  displayName?: string | null;

  settings: {
    profileVisibility: 'private';
    futureSocialEnabled: boolean;
  };

  createdAt: string;
  updatedAt: string;
}
```

Notes:

* use `username` as the login identity
* unique index on `username`
* no email field required in v1 unless you want it later

---

## `sessions`

```ts
interface Session {
  _id: string;
  userId: string;
  tokenHash: string;
  expiresAt: string;
  createdAt: string;
  lastUsedAt: string;
  userAgent?: string | null;
  ipAddress?: string | null;
}
```

Notes:

* store a hashed session token, not the raw token
* set an HTTP-only secure cookie with the raw session token
* look up sessions by hashed token server-side

---

## `library_entries`

```ts
type LibraryBucket = 'catalog' | 'wishlist';

type MediaType = 'movie' | 'tv' | 'album' | 'book' | 'game';

type PhysicalFormat =
  | 'blu_ray'
  | 'dvd'
  | 'vhs'
  | 'cd'
  | 'vinyl'
  | 'cassette'
  | 'hardcover'
  | 'paperback'
  | 'switch'
  | 'ps5'
  | 'xbox'
  | 'digital'
  | 'other';

interface LibraryEntry {
  _id: string;
  userId: string;
  mediaRecordId: string;

  bucket: LibraryBucket;
  mediaType: MediaType;

  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;

  notes?: string | null;
  tags: string[];

  createdAt: string;
  updatedAt: string;
}
```

Indexes:

* `{ userId: 1, bucket: 1, mediaType: 1 }`
* `{ userId: 1, createdAt: -1 }`
* `{ userId: 1, tags: 1 }`

Optional uniqueness rule:

* prevent duplicate exact entries with a compound uniqueness strategy such as:

  * `userId + mediaRecordId + bucket + format`
* or allow duplicates if you want multiple owned copies later

For v1, I would prevent obvious duplicates.

---

## `media_records`

```ts
interface ProviderRefs {
  tmdb?: { id: string; mediaKind?: 'movie' | 'tv' };
  musicBrainz?: { id: string };
  discogs?: { id: string };
  openLibrary?: { id: string };
  rawg?: { id: string };
}

interface ExternalRatings {
  imdb?: number | null;
  rottenTomatoes?: number | null;
  tmdb?: number | null;
  metacritic?: number | null;
}

interface MediaRecordBase {
  _id: string;
  mediaType: MediaType;

  title: string;
  sortTitle?: string | null;
  releaseDate?: string | null;
  year?: number | null;

  imageUrl?: string | null;
  summary?: string | null;

  providerRefs: ProviderRefs;
  externalRatings?: ExternalRatings;

  barcodeCandidates?: string[];

  createdAt: string;
  updatedAt: string;
  lastSyncedAt?: string | null;
}
```

### subtype details

```ts
interface MovieMediaRecord extends MediaRecordBase {
  mediaType: 'movie';
  details: {
    runtimeMinutes?: number | null;
    directors?: string[];
    cast?: string[];
    genres?: string[];
  };
}

interface TvMediaRecord extends MediaRecordBase {
  mediaType: 'tv';
  details: {
    seasons?: number | null;
    episodes?: number | null;
    genres?: string[];
    creators?: string[];
  };
}

interface AlbumMediaRecord extends MediaRecordBase {
  mediaType: 'album';
  details: {
    artists: string[];
    label?: string | null;
    trackCount?: number | null;
    releaseCountry?: string | null;
    catalogNumber?: string | null;
  };
}

interface BookMediaRecord extends MediaRecordBase {
  mediaType: 'book';
  details: {
    authors: string[];
    isbn10?: string | null;
    isbn13?: string | null;
    publisher?: string | null;
    pageCount?: number | null;
  };
}

interface GameMediaRecord extends MediaRecordBase {
  mediaType: 'game';
  details: {
    platforms?: string[];
    developers?: string[];
    publishers?: string[];
    genres?: string[];
  };
}

type MediaRecord =
  | MovieMediaRecord
  | TvMediaRecord
  | AlbumMediaRecord
  | BookMediaRecord
  | GameMediaRecord;
```

Indexes:

* `providerRefs.<provider>.id`
* `{ mediaType: 1, title: 1, year: 1 }`
* `{ barcodeCandidates: 1 }`

---

## `provider_cache` (optional but recommended)

Use this to reduce provider calls and protect yourself from rate limits, especially MusicBrainz. MusicBrainz explicitly requires clients not to exceed one request per second. ([MusicBrainz][3])

```ts
interface ProviderCacheEntry {
  _id: string;
  provider: 'tmdb' | 'musicbrainz' | 'discogs' | 'openlibrary' | 'rawg';
  cacheKey: string;
  payload: unknown;
  expiresAt: string;
  createdAt: string;
}
```

---

## `scan_logs` (optional)

```ts
interface ScanLog {
  _id: string;
  userId: string;
  barcode: string;
  matchedMediaType?: MediaType | null;
  matchedProvider?: string | null;
  createdAt: string;
}
```

---

# 6) Metadata normalization rules

The backend must normalize provider responses into one internal shape.

## Example rule set

* every provider result becomes a `NormalizedSearchResult`
* every provider detail response becomes a `NormalizedMediaRecordInput`
* the backend decides whether to:

  * reuse an existing `media_record`
  * update an existing `media_record`
  * create a new `media_record`

## Normalized search result

```ts
interface NormalizedSearchResult {
  provider: 'tmdb' | 'musicbrainz' | 'discogs' | 'openlibrary' | 'rawg';
  providerId: string;
  mediaType: MediaType;

  title: string;
  subtitle?: string | null;
  year?: number | null;
  imageUrl?: string | null;
  summary?: string | null;

  creatorLine?: string | null;
  barcodeCandidates?: string[];

  confidence?: number | null;
}
```

## Deduplication strategy

Do not over-engineer dedupe in v1.

Use:

* provider ID exact match first
* then fallback heuristic:

  * same `mediaType`
  * normalized title
  * same year if available
  * same primary creator if available

If confidence is ambiguous, create a new `media_record` instead of silently merging.

---

# 7) Auth design

## Requirements

* local username/password only
* password hash stored, never plaintext
* HTTP-only session cookie
* CSRF-safe approach for mutating routes
* rate-limit login attempts
* protect all app pages by default

## Recommended approach

* hash passwords with a memory-hard algorithm
* store hashed session tokens
* rotate session on login
* support logout by deleting the server-side session row

## Routes

* `POST /auth/register`
* `POST /auth/login`
* `POST /auth/logout`
* `GET /auth/me`

## Register rules

* username 3–32 chars
* normalized lowercase username for uniqueness
* strong password minimum
* create default settings object
* immediately create a session on successful register

## Login flow

1. validate input
2. look up user by username
3. verify password hash
4. create session
5. set cookie
6. return current user summary

## Route guarding

* backend guard validates session cookie
* frontend loads `/auth/me` on app boot
* unauthenticated users are redirected to login

---

# 8) API design

Use REST for clarity.

## Auth

* `POST /auth/register`
* `POST /auth/login`
* `POST /auth/logout`
* `GET /auth/me`

## Library

* `GET /library`

  * filters: `bucket`, `mediaType`, `tag`, `search`, `page`
* `POST /library`
* `GET /library/:entryId`
* `PATCH /library/:entryId`
* `DELETE /library/:entryId`

## Media

* `GET /media/:mediaRecordId`
* `POST /media/import`

  * import a normalized provider result into `media_records`
* `POST /media/refresh/:mediaRecordId`

  * optional later

## Search

* `GET /search`

  * query params: `q`, `mediaType`
  * returns aggregated provider results

## Barcode

* `POST /barcode/lookup`

  * body: `{ barcode: string, preferredMediaType?: MediaType }`
  * returns candidate matches

## Health

* `GET /health`
* `GET /health/providers` optional

---

# 9) Barcode flow

## UX

1. user opens scan screen
2. browser requests camera access
3. scanner reads UPC/EAN/ISBN
4. frontend sends barcode to backend
5. backend tries provider lookups in priority order
6. backend returns candidate matches
7. user confirms one
8. app imports or links the record
9. user chooses bucket and optional format/notes/tags

## Provider matching strategy

### Books

* barcode → Open Library ISBN lookup first

### Music

* barcode → Discogs / MusicBrainz candidate flow
* if weak or empty, allow manual search

### Movies / TV

* barcode lookup often weaker with free sources
* fallback to manual confirmation search quickly

### Games

* treat as weakest barcode source
* may rely on barcode-to-search-candidates only if provider supports it
* otherwise fallback to manual search screen pre-filled with scan value

## Critical design rule

Barcode scanning is not “exact match automation.”
It is “input acceleration.”

---

# 10) Frontend pages

## Public

* `/login`
* `/register`

## Protected app

* `/`

  * dashboard / recent additions / quick actions
* `/catalog`
* `/wishlist`
* `/search`
* `/scan`
* `/library/:entryId`
* `/settings`

## Suggested UI behavior

### Dashboard

* counts by media type
* recent additions
* quick links to scan/search/catalog/wishlist

### Catalog / Wishlist

* tabs or separate routes
* filters:

  * media type
  * format
  * tags
  * text search
* card and compact list modes later if needed

### Search

* input text
* media type filter
* show provider results grouped by type/provider
* each result has:

  * image
  * title
  * year
  * creator line
  * “add to catalog”
  * “add to wishlist”

### Scan

* camera viewport
* last scanned code
* candidate results
* retry/manual search option

### Entry detail

Show both:

* user-owned data
* external metadata snapshot

Keep these visually separate:

* “My copy”
* “Media details”

---

# 11) Monorepo structure

```txt
media-library/
  apps/
    web/
      src/
        app/
        components/
        features/
        lib/
      public/
      package.json
    api/
      src/
        main.ts
        app.module.ts
        modules/
          auth/
          users/
          library/
          media/
          providers/
          barcode/
          health/
        common/
        config/
      package.json

  packages/
    types/
      src/
        auth.ts
        library.ts
        media.ts
        provider.ts
        api.ts
      package.json

    provider-sdk/
      src/
        contracts/
        mappers/
        utils/
      package.json

    config/
      eslint/
      typescript/
      package.json

  docker/
    nginx/        # optional later
  docs/
    architecture.md
    api.md
    data-model.md
    deployment.md

  .env.example
  docker-compose.yml
  docker-compose.prod.yml
  package.json
  pnpm-workspace.yaml
  turbo.json
```

---

# 12) Environment variables

## root `.env.example`

```env
NODE_ENV=development

# Web
WEB_PORT=3000
NEXT_PUBLIC_API_BASE_URL=http://localhost:4000

# API
API_PORT=4000
APP_BASE_URL=http://localhost:3000
SESSION_COOKIE_NAME=mlib_session
SESSION_SECRET=replace_me
CORS_ORIGIN=http://localhost:3000

# Mongo
MONGODB_URL=mongodb://mongo:27017/media_library

# TMDB
TMDB_API_KEY=
TMDB_BASE_URL=https://api.themoviedb.org/3

# MusicBrainz
MUSICBRAINZ_BASE_URL=https://musicbrainz.org/ws/2
MUSICBRAINZ_USER_AGENT=media-library/0.1.0 (your-contact-or-repo-url)

# Discogs
DISCOGS_TOKEN=
DISCOGS_BASE_URL=https://api.discogs.com

# Open Library
OPENLIBRARY_BASE_URL=https://openlibrary.org

# RAWG
RAWG_API_KEY=
RAWG_BASE_URL=https://api.rawg.io/api
```

## Dev vs prod notes

* dev uses Docker Mongo
* prod may override `MONGODB_URL` to Atlas
* do not hard-code provider keys
* keep per-service `.env` loading simple

---

# 13) Docker plan

## Development `docker-compose.yml`

Services:

* `web`
* `api`
* `mongo`

### Web

* runs Next dev server

### API

* runs Nest dev server with watch mode

### Mongo

* persistent named volume

## Production `docker-compose.prod.yml`

Services:

* `web`
* `api`
* optional reverse proxy
* Mongo omitted if using Atlas

## Raspberry Pi concerns

* prefer multi-arch-friendly official images
* avoid unnecessarily heavy services
* keep builds simple
* use environment overrides rather than code changes for switching Atlas/local Mongo

---

# 14) Caching and rate-limit strategy

This matters a lot for MusicBrainz. The official docs say each client app must never exceed one call per second. ([MusicBrainz][3])

## Rules

* cache search results by `(provider, mediaType, query)`
* cache detail lookups by `(provider, providerId)`
* cache barcode lookups by `(provider, barcode)`
* use short TTLs for search, longer TTLs for detail
* implement a provider request queue for MusicBrainz
* send a proper User-Agent for MusicBrainz requests

## Suggested TTLs

* search: 12 hours
* detail: 7 days
* barcode lookup: 7 days

These are design defaults, not hard rules.

---

# 15) Validation and security rules

## Input validation

Validate all request bodies and query params.

## Auth security

* hash passwords securely
* hash session tokens before persistence
* use HTTP-only cookies
* set `SameSite=Lax` by default
* use `Secure` in prod
* add login rate limiting
* do not reveal whether username or password was wrong in detail

## Permissions

Every protected route must scope queries to the current `userId`.

## Logging

Log:

* login success/failure
* provider errors
* barcode lookup attempts
* unexpected normalization failures

Do not log:

* passwords
* raw session secrets
* provider keys

---

# 16) Implementation order

## Milestone 1 — Monorepo and scaffolding

* set up pnpm workspace
* set up turbo
* scaffold Next app
* scaffold Nest app
* create shared types package
* add Docker Compose
* connect API to Mongo
* health route works

## Milestone 2 — Auth

* users collection
* sessions collection
* register/login/logout/me
* HTTP-only session cookie
* frontend login/register pages
* route protection

## Milestone 3 — Core library model

* `media_records`
* `library_entries`
* create/list/update/delete entries
* catalog and wishlist pages
* basic filters

## Milestone 4 — Search integrations

* TMDB integration
* Open Library integration
* MusicBrainz integration with queue/rate limiting
* RAWG integration
* Discogs optional after the basics work
* aggregated search endpoint

## Milestone 5 — Import and normalization

* normalize provider results
* create/import `media_records`
* link imported record to `library_entries`
* dedupe rules

## Milestone 6 — Barcode scan flow

* frontend scan page
* backend barcode lookup service
* candidate match UI
* add to catalog/wishlist

## Milestone 7 — Polish

* docs
* seed data
* README
* screenshots
* architecture diagrams
* GitHub-ready cleanup

---

# 17) Acceptance criteria for v1

A v1 is complete when:

1. A user can register and log in.
2. All app routes are protected by default.
3. A user can search movies, TV, albums, books, and games.
4. A user can add results into either catalog or wishlist.
5. A user can edit notes, tags, format, and purchase date.
6. A user can scan a barcode on mobile web and get candidate matches.
7. The app stores normalized metadata and user-specific collection data separately.
8. The app runs locally with Docker Compose.
9. The app can switch from local Mongo to Atlas through environment variables.
10. The repo has strong documentation.

---

# 18) README structure

Use this structure:

## Title

Media Library Manager

## One-line summary

A self-hostable full-stack TypeScript app for managing personal collections across movies, TV, music, books, and games.

## Why I built it

Focus on:

* portfolio quality
* API integration
* data normalization
* barcode scanning
* Docker/self-hosting
* backend architecture

## Features

* login + session auth
* catalog + wishlist
* metadata search from public providers
* barcode-assisted lookup
* mobile-friendly UI
* Dockerized setup
* MongoDB-backed persistence

## Architecture

* Next.js frontend
* NestJS backend
* MongoDB
* provider adapter pattern
* shared TypeScript types

## External providers

* TMDB
* MusicBrainz
* Open Library
* RAWG
* Discogs optional enrichment

Mention attribution where required, especially TMDB and RAWG. ([The Movie Database (TMDB)][2])

## Local setup

* env copy
* Docker Compose up
* URLs
* seed if present

## Screenshots

* login
* search
* catalog
* scan flow
* item detail

## Future improvements

* shared/public profiles
* better edition modeling
* PWA
* import/export
* richer music/game coverage

---

# 19) GitHub/job-hunt framing

Use wording like this in your repo description:

> Full-stack TypeScript media collection manager with NestJS, Next.js, MongoDB, provider-based metadata ingestion, and mobile-web barcode scanning.

Use wording like this in your resume or portfolio:

> Built a self-hostable media library platform with a separated Next.js/NestJS architecture, MongoDB persistence, secure session auth, third-party metadata integrations, caching, and barcode-assisted mobile lookup for physical media cataloging.

That framing highlights:

* architecture
* backend ownership
* API work
* product thinking
* deployability

---

# 20) Master Cursor prompt

Copy this into Cursor as the main build brief.

```txt
You are helping build a production-minded portfolio project called “Media Library Manager.”

Goal:
Create a self-hostable full-stack TypeScript web application for managing personal collections across movies, TV shows, music albums, books, and games.

Core product requirements:
- Private, login-first web app
- Multi-user aware architecture, even though the primary use case is personal/self-hosted
- Each user has exactly two buckets in v1: catalog and wishlist
- Ratings are read-only external metadata only when a provider exposes them
- No user review/rating system in v1
- Mobile-web barcode scanning with best-effort candidate matching
- Focus on physical media, especially albums
- Strong backend architecture is more important than flashy frontend polish
- Clear frontend/backend separation
- Monorepo
- TypeScript everywhere
- Dockerized local and production flows
- Dev uses Docker MongoDB
- Prod can switch to MongoDB Atlas via environment variables
- No email verification or password reset flow
- Secure password hashing and session-based auth required

Architecture requirements:
- Frontend: Next.js + TypeScript + Tailwind
- Backend: NestJS + TypeScript
- Database: MongoDB
- Monorepo with apps/web, apps/api, packages/types, packages/provider-sdk, packages/config
- Frontend must never call third-party metadata providers directly
- Backend owns auth, sessions, provider integrations, normalization, barcode lookup, caching, and permission checks

Data model requirements:
Use these collections:
- users
- sessions
- library_entries
- media_records
- optional provider_cache
- optional scan_logs

Do NOT model this as SQL-style inheritance tables like item_as_cd or item_as_game.

Data separation rules:
1. library_entries stores user-specific collection state
   - userId
   - mediaRecordId
   - bucket (catalog|wishlist)
   - mediaType
   - format
   - barcode
   - purchaseDate
   - notes
   - tags
   - createdAt
   - updatedAt

2. media_records stores normalized provider metadata
   - mediaType
   - title
   - sortTitle
   - releaseDate
   - year
   - imageUrl
   - summary
   - providerRefs
   - externalRatings
   - barcodeCandidates
   - lastSyncedAt
   - subtype-specific details object keyed by mediaType

Subtype detail examples:
- movie: runtime, directors, cast, genres
- tv: seasons, episodes, creators, genres
- album: artists, label, trackCount, releaseCountry, catalogNumber
- book: authors, isbn10, isbn13, publisher, pageCount
- game: platforms, developers, publishers, genres

External providers:
- TMDB for movies/TV
- MusicBrainz for music
- Discogs as optional enrichment for music release/barcode details
- Open Library for books
- RAWG for games, but keep games provider replaceable behind an adapter

Provider rules:
- Store normalized internal records, not a full mirror of third-party APIs
- Keep provider IDs for refresh
- Cache provider responses
- MusicBrainz integration must be queue/rate-limit aware
- Keep provider implementations behind clear interfaces

Auth requirements:
- username + password only
- secure password hashing
- hashed session tokens in database
- httpOnly session cookie
- register/login/logout/me routes
- protected routes by default

API requirements:
Implement these route groups:
- auth
- library
- media
- search
- barcode
- health

Suggested routes:
POST /auth/register
POST /auth/login
POST /auth/logout
GET /auth/me

GET /library
POST /library
GET /library/:entryId
PATCH /library/:entryId
DELETE /library/:entryId

GET /media/:mediaRecordId
POST /media/import
POST /media/refresh/:mediaRecordId

GET /search?q=&mediaType=
POST /barcode/lookup
GET /health

Barcode behavior:
- Mobile-web camera scanning
- UPC/EAN/ISBN style codes
- Scan returns candidate matches
- User confirms one result before import
- Barcode flow should be especially good for books and music
- Movies/TV/games can fall back to text search when barcode matching is weak

Frontend pages:
- /login
- /register
- /
- /catalog
- /wishlist
- /search
- /scan
- /library/:entryId
- /settings

UI expectations:
- Clean, presentable, mobile-friendly
- Functional over fancy
- Separate “My copy” info from imported “Media details”
- Filters for media type, tags, and search

Engineering expectations:
- Clear folder structure
- Shared TypeScript types
- Good naming
- Strong validation
- Good error handling
- Seed data if useful
- Strong README
- Architecture docs
- Docker Compose setup
- Environment variable example file

Build incrementally.
Before generating code for each major part:
1. restate the target
2. list affected files
3. implement complete working code
4. explain any assumptions briefly
5. avoid placeholder pseudocode unless explicitly requested

Start by generating:
1. the monorepo folder structure
2. the root package.json / workspace / turbo config
3. docker-compose for dev
4. NestJS API scaffolding plan
5. Next.js web scaffolding plan
6. shared types package plan
```

---

# 21) Milestone prompts for Cursor

## Prompt 1 — scaffold the monorepo

```txt
Set up the initial monorepo for the Media Library Manager project.

Requirements:
- pnpm workspace
- turbo repo
- apps/web for Next.js
- apps/api for NestJS
- packages/types for shared TypeScript types
- packages/provider-sdk for provider contracts and mapping utilities
- packages/config for shared tsconfig/eslint settings
- root scripts for dev, build, lint, typecheck
- docker-compose.yml for local development with web, api, and mongo
- .env.example

Output:
- folder tree
- all root config files
- package.json files
- minimal working scripts
- no skipped files
```

## Prompt 2 — build auth

```txt
Implement authentication for the Media Library Manager project.

Requirements:
- NestJS backend
- MongoDB collections: users, sessions
- username/password auth only
- secure password hashing
- hashed session tokens in DB
- httpOnly cookie session auth
- endpoints:
  POST /auth/register
  POST /auth/login
  POST /auth/logout
  GET /auth/me
- Next.js pages:
  /login
  /register
- protect app routes by default
- create auth guard/middleware for backend
- create frontend auth bootstrap flow
- shared request/response types in packages/types

Output complete working code, including:
- DTOs / validators
- controllers
- services
- repositories or db access layer
- cookie/session handling
- frontend forms and route protection
```

## Prompt 3 — library data layer

```txt
Implement the core library domain.

Requirements:
- Mongo collections:
  - media_records
  - library_entries
- shared domain types for media records and library entries
- CRUD endpoints for library entries
- filters by bucket, mediaType, tags, search text
- prevent obvious duplicates for the same user/bucket/mediaRecordId/format
- Next.js pages for:
  - /catalog
  - /wishlist
  - /library/[entryId]
- separate “My copy” fields from imported metadata fields in the UI

Output complete code and keep the domain clean and typed.
```

## Prompt 4 — provider abstraction

```txt
Implement the provider abstraction layer.

Requirements:
- provider contract interfaces in packages/provider-sdk
- adapters for:
  - TMDB
  - MusicBrainz
  - Open Library
  - RAWG
- Discogs adapter can be included as optional enrichment
- normalized search result type
- normalized media import type
- backend search service that fans out to providers by media type
- backend caching interface for provider responses
- MusicBrainz request throttling / queue awareness
- no frontend direct provider calls

Create clean interfaces and complete provider service implementations with environment-based configuration.
```

## Prompt 5 — search and import flow

```txt
Implement the search and import flow.

Requirements:
- GET /search?q=&mediaType=
- aggregate provider results into a normalized response
- allow frontend search page to show results
- POST /media/import should:
  - accept a normalized provider result or provider reference
  - fetch/normalize provider details if needed
  - dedupe or create media_record
  - optionally create a library_entry in catalog or wishlist
- create the /search page
- allow add-to-catalog and add-to-wishlist actions from search results

Keep provider metadata and user-owned library state clearly separated.
```

## Prompt 6 — barcode flow

```txt
Implement barcode scanning.

Requirements:
- mobile-web scanning page at /scan
- camera-based barcode scanning in browser
- support UPC/EAN/ISBN style scanning
- POST /barcode/lookup backend route
- barcode lookup orchestrator that tries providers in sensible priority order
- return candidate matches, never assume exact automatic match
- allow user to confirm a candidate and save it to catalog or wishlist
- if no strong match exists, offer fallback to manual search with prefilled query

Output complete frontend and backend code with error handling and graceful fallback behavior.
```

## Prompt 7 — polish and docs

```txt
Polish the project for GitHub and job-hunting.

Requirements:
- improve README
- add architecture.md
- add api.md
- add data-model.md
- add deployment.md
- add seed data or demo fixtures if useful
- clean up naming and file structure
- add health checks
- add error boundary or equivalent UI handling
- add loading/empty states
- ensure env files and setup instructions are clear
- ensure Docker instructions are complete

The output should make the repo feel like a serious portfolio project, not a hackathon prototype.
```

---

# 22) Practical implementation notes for the AI

Tell Cursor these rules if it starts drifting:

* Do not collapse everything into a single app.
* Do not use NextAuth/Auth.js.
* Do not use Prisma.
* Do not replace MongoDB with Postgres.
* Do not introduce email flows.
* Do not introduce social features in v1.
* Do not over-model editions/releases unless necessary.
* Do not let the frontend call TMDB/MusicBrainz/Open Library/RAWG directly.
* Do not store raw third-party provider payloads everywhere by default.
* Do not assume barcode scan equals exact match.
* Do not put all types inside app-local folders; use shared packages.

---

# 23) Final decisions to hard-code

Use these exact assumptions in the generated code:

* single repo, multi-app monorepo
* Next.js frontend
* NestJS backend
* MongoDB
* local username/password auth
* session cookie auth
* two buckets only: catalog and wishlist
* media types: movie, tv, album, book, game
* games provider behind adapter; RAWG first
* mobile-web barcode scanning
* private app by default
* physical-media-oriented formats
* backend-first architecture quality
* Dockerized local development
* Raspberry Pi friendly prod setup
* Atlas-compatible prod config

---

# 24) Best next step

Start with the **master prompt**, then use the milestone prompts one by one. That gives you the best balance between consistency and control.

If you want, I can also turn this into a tighter **“single paste, zero ambiguity” Cursor spec** that is shorter and more forceful for actual implementation.

[1]: https://www.mongodb.com/pricing?utm_source=chatgpt.com "MongoDB Pricing"
[2]: https://developer.themoviedb.org/docs/faq?utm_source=chatgpt.com "FAQ - TMDb API"
[3]: https://musicbrainz.org/doc/MusicBrainz_API?utm_source=chatgpt.com "MusicBrainz API"
[4]: https://openlibrary.org/developers/api?utm_source=chatgpt.com "APIs"
[5]: https://api.rawg.io/docs/?utm_source=chatgpt.com "RAWG Video Games Database API"
