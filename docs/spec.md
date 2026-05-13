# Media Library Manager — Application specification

## Overview

This repository ships **one Go web application** at the repository root (`cmd/web`, `internal/...`). It is a private, login-first media library with MongoDB persistence and server-side metadata providers.

Authoritative persistence details: [`docs/data-model.md`](data-model.md).

Historical note on the retired Node monorepo: [`docs/migration-from-legacy.md`](migration-from-legacy.md).

---

## Project goals

### Primary

- Single deployable binary and straightforward operations
- Self-hosting friendly (including Raspberry Pi–class hardware)
- Core catalog behavior: multiple media types, catalog vs wishlist, provider-backed discovery
- Codebase that reads as an intentional Go application, not a translated client-heavy SPA

### Secondary

- Clear layering for portfolio discussion (handlers, services, repositories)
- Room to extend without splitting into separate frontend/backend services

---

## Product scope

Media types: **movie**, **tv**, **album**, **book**, **game**.

Each user has exactly two buckets:

- `catalog` — owned / tracked items
- `wishlist` — wanted items

The app is private by default, multi-user capable, and oriented toward personal/self-hosted use.

---

## Core product rules

- Users must authenticate before app functionality.
- Ratings are **read-only** external metadata (never user-authored scores as product truth).
- Barcode lookup is **best-effort** candidate matching; it **must not** auto-save library state.
- **`media_records`** (canonical metadata) and **`library_entries`** (user-owned rows) stay **separate** documents; do not collapse them.
- Canonical metadata and provider normalization are **server-owned**; the browser does not call TMDB, MusicBrainz, Open Library, RAWG, or Discogs directly.
- UI presentation is localized: **English** and **Japanese**; internal enums and stored values stay stable (not translated).

---

## Non-goals (v1 scope)

- Native mobile app, public/social features
- Email verification, password reset, OAuth
- Distributed microservice deployment as the default story
- A React/Vue-style SPA shipped as the primary UI
- Heavy edition/pressing modeling

---

## Technical stack

| Concern | Choice |
|--------|--------|
| Runtime | Go |
| HTTP | `net/http`, routing via `chi` |
| HTML | `html/template`, server-rendered pages by default |
| Enhancement | htmx for selective partial HTML; small vanilla JS where the browser requires it (e.g. scan flow) |
| Data | MongoDB, official Go driver |
| Production assets | `go:embed` for templates, locales, and static files when `APP_ENV=production` |

---

## Architecture style

The app is a **monolith**: routing, auth, sessions, HTML rendering, localization, providers, import/dedupe, barcode orchestration, and persistence live in this one codebase. There is no separate API-only service or Next-style frontend app in this repo.

---

## Rendering model

### Default

Full pages are rendered on the server from Go templates.

### Progressive enhancement (htmx)

**htmx** is used where **partial HTML** responses improve responsiveness without changing who owns state. Examples in this codebase: swapping search results, paginating search or library lists, filter-driven list refresh, inline notices after import, and inline feedback on media refresh. The same routes often return a **full page** for a normal request and a **fragment** when the request is an htmx request (see handlers and templates).

### JavaScript

**Vanilla JS** is used where it clearly helps UX or is required by the platform: barcode scanning (camera lifecycle, ZXing), and other browser-only behavior. The app is **not** “no-JS”; it is **HTML-first** with **JS-enhanced** flows where appropriate.

### Architectural rule

The **server** remains the source of truth: business rules, canonical metadata, and authoritative state stay in Go. Do not recreate SPA-style global client state in templates or ad hoc JS. Do not treat the browser as the owner of canonical catalog data.

---

## Routing model

User-facing **pages** use a **locale prefix**: `en` or `ja` (e.g. `/en/catalog`, `/ja/search`). `/` redirects to the configured default locale.

Some **mutations** use stable **non-locale** paths (e.g. `POST /library`, `POST /media/import`) to keep forms simple; after success, redirects preserve locale. See [`docs/routes.md`](routes.md) and [`docs/api.md`](api.md).

---

## Authentication

- Username / password
- Password hashing via `golang.org/x/crypto`
- Server-side sessions stored in MongoDB; HTTP-only session cookie
- No JWT-in-localStorage, no OAuth, no email flows for auth

---

## Domain model (summary)

Collections: `users`, `sessions`, `media_records`, `library_entries`; optional `provider_cache`, `scan_logs`. Canonical media fields live in `media_records`; per-user notes, tags, bucket, format, barcode, etc. live in `library_entries`. Details: [`docs/data-model.md`](data-model.md).

---

## Providers

Supported: TMDB, MusicBrainz, Discogs, Open Library, RAWG. Calls are server-side only; responses are normalized; caching and MusicBrainz throttling are configurable. Raw provider payloads are not the application schema. Operational detail: [`docs/provider-notes.md`](provider-notes.md).

---

## Search

Search is server-driven; results are **transient** until the user runs an explicit import/save flow. Client-side enhancement (htmx) may refresh result markup and pagination without a full page reload.

---

## Import and dedupe

Import is **backend-owned**: provider identity → normalize → dedupe → create or reuse `media_record` → optional `library_entry`. The browser does not own canonical metadata.

---

## Barcode

Normalize input; prefer local matches where useful; ordered provider stages; candidates plus explicit fallback guidance. Books and albums are the strongest cases; movies, TV, and games may be weaker. Never auto-import from lookup alone.

---

## UI principles

- HTML-first, mobile-aware, functional over flashy
- Explicit save / import actions
- Detail pages emphasize **My copy** vs **media details** (user row vs canonical metadata)

---

## Reliability

Provider cache (Mongo-backed), TTLs, MusicBrainz rate limiting, structured errors, and normalized contracts reduce duplicate upstream traffic and keep partial failures from corrupting data.

---

## Deployment

Local dev and production Docker paths are documented in [`docs/deployment.md`](deployment.md). High level: development often runs Mongo in Docker and the Go app on the host; production builds a static Linux binary with embedded assets when `APP_ENV=production`.

---

## Success criteria (maintainer view)

- One Go application clearly owns product behavior
- Deployment is simpler than a split frontend/backend monorepo
- New contributors can follow `docs/architecture.md`, this spec, and `internal/app/app.go` without migration-era context
