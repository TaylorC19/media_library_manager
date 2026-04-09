# Media Library Manager — Go Rewrite Spec

## Overview

This document defines the target architecture for the Go rewrite of Media Library Manager.

The repository currently contains an older implementation in:

- `apps/web`
- `apps/api`

Those directories are the **reference implementation only**.

The goal of this rewrite is to fully replace that implementation with a **single Go application** at the repository root.

This rewrite should not result in a permanent side-by-side structure such as a long-lived `go-v2/` app. Temporary coexistence is allowed during migration, but the intended final state is a single Go-based application that owns the product.

---

## Rewrite Goals

### Primary goals
- Replace the current Next.js/NestJS implementation with a Go application
- Simplify deployment and runtime overhead
- Make the application easier to self-host on a Raspberry Pi
- Preserve the core product behavior of v1
- Build a codebase that reads like an intentional Go application, not a translated SPA

### Secondary goals
- Improve backend and systems programming depth in the project
- Keep the architecture easy to explain in a portfolio
- Preserve room for future iteration and extension

---

## Final Architecture Goal

The final repository should become a single Go web app using:

- Go
- `net/http`
- `chi`
- `html/template`
- small vanilla JS
- optional htmx
- MongoDB
- embedded templates/assets in production where appropriate

The final repository should **not** retain:

- the old `apps/web` and `apps/api` structure
- a permanent parallel app layout
- a frontend/backend split as separate applications

---

## Migration Intent

### Current repo reality
The repo currently contains the old app.

### Migration rule
During migration:
- the old app may be read as reference
- the old app should not receive new feature investment
- new architecture work should move toward the final Go structure at the repo root
- old code may be removed intentionally once replacement coverage exists

### Important constraint
Do not design the rewrite as a permanent “v1 + v2” multi-app repository.

---

## Product Scope

Media Library Manager manages personal collections across:

- movies
- TV shows
- music albums
- books
- games

The application remains:

- private/login-first
- multi-user capable
- primarily self-hosted/personal-use oriented
- physical-media-friendly

Each user has exactly two buckets:

- `catalog`
- `wishlist`

---

## Core Product Rules

- The app is private by default.
- Users must log in before accessing app functionality.
- Ratings are read-only external metadata only.
- Barcode lookup is best-effort candidate matching only.
- Barcode lookup must never auto-save.
- `media_records` and `library_entries` must remain separate.
- Canonical metadata remains backend-owned.
- External providers are server-side only.
- English and Japanese are supported.

---

## Non-Goals

These are out of scope for the initial Go rewrite:

- native mobile app
- public/social features
- email verification
- password reset
- OAuth
- media playback
- distributed infrastructure
- a React/Vue/SPA frontend
- over-engineered edition modeling

---

## Technical Stack

### Core application
- Go
- `net/http`
- `chi`

### Rendering
- `html/template`
- server-rendered HTML
- optional htmx for partial updates
- small vanilla JS only where necessary

### Persistence
- MongoDB
- official MongoDB Go Driver

### Packaging / deployment
- `embed`
- Docker
- Raspberry Pi-friendly deployment model

---

## Architecture Style

The rewrite is a **Go monolith**.

The Go application owns:

- routing
- auth
- session handling
- HTML rendering
- localization
- provider integrations
- import/dedupe logic
- barcode orchestration
- persistence

This is intentionally different from the old split architecture.

---

## Rendering Model

### Default
Render full pages on the server.

### Progressive enhancement
Use htmx where partial updates clearly improve UX.

### Browser-side JS
Use small vanilla JS only where the browser requires it, especially for:

- camera access
- barcode scanning lifecycle

### Anti-goals
Do not reproduce SPA-style client state architecture in templates.

---

## Routing Model

Use locale-prefixed HTML routes such as:

- `/en/login`
- `/ja/login`
- `/en/catalog`
- `/ja/catalog`

Root `/` should redirect to a default or detected locale.

Protected pages should preserve locale context.

Some action routes may remain non-locale endpoints if that simplifies form submission and redirects, but page rendering must preserve locale.

---

## Authentication Model

Use:

- username/password
- secure password hashing
- server-side sessions
- HTTP-only cookies
- protected routes by default

Do not use:

- JWT stored in browser local storage
- OAuth
- email flows

---

## Domain Model

The rewrite preserves the core v1 domain model.

### Collections
- `users`
- `sessions`
- `media_records`
- `library_entries`

Optional:
- `provider_cache`
- `scan_logs`

### Separation rule
Canonical media metadata must remain separate from user-owned library state.

---

## Domain Separation Rules

### `media_records`
Stores:
- title
- year
- summary
- image
- provider refs
- ratings
- barcode candidates
- subtype details
- sync timestamps

Does not store:
- notes
- tags
- purchase date
- bucket
- format

### `library_entries`
Stores:
- userId
- mediaRecordId
- bucket
- format
- barcode
- purchase date
- notes
- tags

Does not store:
- canonical provider metadata as primary state
- raw provider payloads

---

## Provider Rules

Supported providers remain:

- TMDB
- MusicBrainz
- Discogs
- Open Library
- RAWG

Rules:
- provider calls are server-side only
- all provider results are normalized
- provider caching is allowed
- MusicBrainz throttling remains explicit
- provider failures must not corrupt data
- raw provider payloads are not the internal schema

---

## Search Rules

Search is server-rendered first.

It may later be enhanced with htmx for:
- result refresh
- filtering
- pagination

Search results are transient and not canonical stored records by themselves.

---

## Import Rules

Import remains backend-owned.

The import pipeline must:
1. accept provider-based identity
2. fetch details when needed
3. normalize provider data
4. dedupe safely
5. create or reuse `media_record`
6. optionally create `library_entry`

The frontend must not become the source of truth for canonical metadata.

---

## Barcode Rules

Barcode scanning is a helper flow, not an exact identification guarantee.

The system should:
- scan UPC/EAN/ISBN-style codes
- normalize barcode input
- check local matches first
- run provider lookups in order
- return candidates plus explicit fallback guidance

Books and albums are the strongest barcode cases.  
Movies, TV, and games may remain weaker/manual-fallback cases.

---

## Localization

Supported languages:
- English
- Japanese

Rules:
- locale-prefixed page routes
- translated UI strings
- internal enum/database values remain stable
- provider metadata is not translated by default
- locale persists through navigation and redirects

---

## UI Principles

- HTML-first
- mobile-first
- functional over flashy
- form-first flows
- explicit save actions
- clean separation of user-owned and canonical metadata

Important detail-page split:
- **My copy**
- **Media details**

---

## Reliability Principles

- Mongo-backed provider cache
- configurable TTLs
- explicit MusicBrainz throttling
- reduced repeated provider calls
- structured provider errors
- stable normalized response contracts

---

## Deployment Principles

- local development should be straightforward
- production should be easy to self-host
- Raspberry Pi should be a first-class deployment target
- local Mongo vs Atlas should be easy to switch
- the final Go app should be simpler to deploy than the current split app

---

## Final-State Success Criteria

The rewrite is successful when:

- the Go app fully covers the intended product behavior
- the old app is no longer needed
- the repo centers on a single Go application
- deployment is simpler than the old architecture
- the codebase reads clearly as an HTML-first Go application