# Media Library Manager Alpha Version 0.1.0

A self-hosted media library app for tracking movies, TV, albums, books, and games in one private collection.

`Media Library Manager` is built for people who want to keep their own catalog instead of spreading it across spreadsheets, notes, and closed third-party services. The app is a **single Go web application** (server-rendered HTML) with MongoDB persistence and server-side metadata providers.

In v1, every item lives in one of two buckets:

- `catalog` for things you already own
- `wishlist` for things you want to track without mixing them into the main collection

## Live demo

[Open the live app](https://media-library-prod.vercel.app/).

Public demo account:

- **Username:** `demo`
- **Password:** `password`

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
- Admin, general, and demo/read-only roles; full-database backups require admin access
- My Library landing page with Collection and Wishlist views
- Movies, TV, albums, books, and games
- Manual entry creation when provider data is missing or incomplete
- Provider-backed search with normalized results
- Search with title lookup, barcode entry, and camera scanning, with explicit confirmation before saving
- Personal provider API keys configurable from Settings, with environment defaults
- English and Japanese UI support
- Docker Compose installation with authenticated MongoDB, automatic database setup, and administrator bootstrap
- `GET /health` for readiness and container health checks
- Full-database maintenance (development Settings UI, `cmd/db`, and `mongodump` scripts — see [`docs/backup.md`](docs/backup.md))

## Philosophy

- the browser never talks to third-party metadata providers directly
- the Go server owns auth, provider access, normalization, caching, and barcode lookup
- user-owned collection data is stored separately from imported metadata
- barcode scanning is treated as candidate matching, not silent automation

## Why Go

A single Go binary keeps deployment small and predictable: one process to run on a home server or a Raspberry Pi, explicit layering (handlers → services → repositories), and strong control over concurrency and resource use. Provider HTTP calls, normalization, caching, and sessions stay in one codebase—no separate “frontend” and “API” deployables to version together.

## Why HTML-first (and not a SPA)

Pages are rendered on the server with `html/template`. That makes auth, links, and forms straightforward, works well for a private catalog app, and keeps **canonical state on the server**. The UI is **not** a single-page React or Vue app; it also is **not** “no JavaScript.” **[htmx](https://htmx.org/)** swaps in HTML fragments for search results, list filters, pagination, and inline notices so interactions feel responsive without moving business logic into the browser. **Vanilla JS** is used where the platform requires it—especially the **scan** flow (camera and barcode decoding). See [`docs/templates.md`](docs/templates.md).

## Quick Start (Docker Compose)

Download the matching **Docker image bundle** from [the alpha release](https://github.com/TaylorC19/media_library_manager/releases/tag/v0.1.0-alpha.1).
It includes the app and setup images, Compose files, and installation guide.
Choose `linux_amd64` or `linux_arm64` for Docker's architecture, including when
using Docker Desktop on macOS/Windows. Verify the archive against `SHA256SUMS`,
extract it, and run from the extracted directory (macOS/Linux shell):

```sh
docker load --input images.tar
docker run --rm --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/config" \
  media-library-manager-setup:0.1.0-alpha.1 init \
  -dir /config -origin https://library.example.com
docker compose --env-file .env.prod up -d
```

Replace the origin with your HTTPS address and configure your reverse proxy to
forward to `127.0.0.1:8080`. Setup generates private credentials, initializes the
bundled database, and creates the first `admin` account. Its random password is
in the owner-only `.bootstrap-admin-password` file; no passwords are printed.
Keep generated files and database volumes backed up privately. Repeating startup
preserves saved data and existing passwords.

The web app starts after the setup service succeeds. No source build, registry login, local Go installation, manual collection
creation, or GitHub Actions setup is required. See the
[installation guide](docs/deployment.md) for updates, external databases, and
Vercel, or the [packaged setup tool](docs/setup-tool.md) for standalone binaries.

For development with Go on your host, use `.env.example`,
`./scripts/dev-db-up.sh`, and `./scripts/dev-web.sh`. The default
`docker-compose.yml` remains the development database-only stack.

## Configuration

Key variables are documented in [`.env.example`](.env.example) and loaded in [`internal/config/config.go`](internal/config/config.go). They include MongoDB, session cookie settings, provider API keys, MusicBrainz throttling, and provider cache TTLs.

Templates:

- `.env.example` — local development
- `.env.setup.example` — private maintenance configuration for an external database
- `.env.prod.example` — runtime configuration reference for external databases (see [`docs/deployment.md`](docs/deployment.md))

## Providers

- **TMDB** — movies and TV
- **MusicBrainz** — albums
- **Discogs** — music enrichment and some barcode support
- **Open Library** — books
- **RAWG** — games

Details: [`docs/provider-notes.md`](docs/provider-notes.md).

## Deployment

[Docker Compose is the primary installation path](docs/deployment.md).
Vercel is an optional managed deployment: initialize its external database with
the Docker setup image or a packaged `mlm-setup` executable, then deploy the Docker
web app using the checked-in Vercel Container configuration. Installers do not need to fork this repository or run GitHub Actions.

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
docker-compose.yml development DB; docker-compose.prod.yml complete self-hosted installation
cmd/setup/         packaged installer (mlm-setup)
```

## Roadmap

- richer settings and account management
- per-user library export (subset without auth data)
- saved filters and collection views
- deeper edition and release modeling for physical media
- background metadata refresh
- more deployment examples and reviewed screenshots using synthetic data; keep raw captures private

## Account roles

When explicitly enabled, public registration creates **general** accounts. Production defaults to admin-created accounts. Existing accounts without a role also remain general; none are automatically promoted. **Admin** can manage users, saved provider keys, and their own library. Web database maintenance is available only in explicitly enabled development installations. **General** can manage their own library and provider keys. **Demo** can browse, search, and test provider connections, but cannot save library/settings changes, refresh metadata, or use database backups. Demo barcode lookups do not create scan-log records.

Fresh installations bootstrap the initial administrator through the setup tool.
For developer recovery, promote an existing account using trusted server access:

```bash
go run ./cmd/users -username YOUR_USERNAME -role admin
```

Production Compose includes the same management command:

```bash
docker compose -f docker-compose.prod.yml exec app ./users -username YOUR_USERNAME -role admin
```

Admins can open **User management** from the header or Settings to search accounts, create accounts, and change other users’ roles. Your own admin role is disabled to prevent lockout. Role changes require current administrator access; the server-side recovery command remains available.

Each account also has **Manage saved keys**. Admins can see which providers have personal keys saved, without viewing their values. Clearing a key requires a separate confirmation page and checkbox. It removes only that account’s selected provider override: the server default applies afterward if configured. It does not revoke the credential with the provider or change other accounts’ keys.

The developer/server command remains available for recovery. Use `-role general` or `-role demo` to change access later. Changes apply on the next authenticated request, including existing sessions. There is no public role-selection or self-promotion endpoint. Older database backups without roles restore accounts with general access; use the command again to promote the intended administrator.

## Improvement backlog

Tracked improvements; completed items are checked below. User management is also implemented.

- [x] Rich saved-item pages: cover, description, creators, and media-specific details, separate from copy information.
- [x] Collection/Wishlist badges in search, with existing saves clearly indicated.
- [x] Direct Wishlist → Collection action that preserves notes and copy details.
- [x] Progressive grouped search: show each group as it finishes.
- [x] Actual provider pagination beyond the initial fetched batch.
- [x] Persistent layout/filter preferences across visits (per account in the current browser).

### Next improvements and browser QA

Live browser review: [2026-10-30 sanitized QA report](docs/qa/2026-10-30/report.md). Confirmed defects are separate from improvement suggestions; destructive and account-changing flows were not submitted.

- [x] **BQA-01 — Responsive detail layout:** stack cover/metadata by component width and stack metadata labels/values on narrow cards; verified in the browser at 390, 515, and 1280 px.
- [x] **I-01 — Save and continue browsing:** save inside the search dialog, update badges immediately, and offer Open saved item for existing saves.
- [x] **I-02 — Deletion confirmation:** show the exact item title and copy details before deleting, with Cancel and server-enforced confirmation. Trash/Undo remains a possible future improvement.
- [x] **I-03 — Sorting and discovery:** title/year/recently-added sorting, private tag filters, clickable tags, and creator/tag search.
- [x] **I-04 — Better copy details:** media-appropriate formats in forms/filters, optional format and notes when saving from Search, and preservation of existing copies and legacy formats.
- [x] **I-05 — Rich manual entries:** editable cover image URLs, descriptions, and creators for manual items, including safe edits of shared manual records.
- [ ] **I-06 — Personal import/export:** account-scoped CSV/JSON export and import with a preview and duplicate handling.
- [ ] **I-07 — Bulk actions:** select multiple items to tag, update formats, or move to Collection, with per-item conflict reporting.
- [ ] **I-08 — Account self-service:** change display name/password and provide an admin recovery flow.
- [ ] **I-09 — Large-library queries:** move Library filtering/sorting/pagination into database queries; current scalability concern is code-derived, not a measured browser slowdown.
- [x] **I-10 — Unsaved-edit protection:** warn before leaving changed Library, provider settings, admin user, and search-dialog forms; successful saves clear the saved changes. Local drafts remain a possible future improvement.
- [ ] **I-11 — Compact mobile browsing:** reduce the search header/control footprint and add media-type jump links or collapsible groups.
