# Media Library Manager — Go Rewrite Milestones

## Overview

This document defines the migration milestones for replacing the current application with a single Go web app.

The repository currently contains an older implementation in:

- `apps/web`
- `apps/api`

These directories are the **reference implementation only** during migration.

The final goal is:

- one Go application
- root-level Go app structure
- old app removed once replacement coverage exists

This is not a permanent side-by-side architecture plan.

---

## Migration Principles

### Temporary coexistence only
The old app may coexist during migration, but only temporarily.

### Reference-only rule
The old implementation may be used for:
- behavior reference
- route reference
- data model reference
- UX reference

It should not keep receiving new feature investment.

### Final-state rule
Each milestone should move the repo closer to:
- root-level Go application
- removal of old app code
- simplified final repository structure

---

## Milestone 1 — Establish the New Go App Foundation

### Goal
Create the new Go application foundation at the repo root.

### Tasks
- initialize Go module at repo root
- create:
  - `cmd/web/main.go`
  - `internal/...`
- add config loading
- add MongoDB connection
- add `chi`
- add base middleware:
  - logging
  - recovery
  - locale
- add static file serving
- add template rendering foundation
- define `embed` strategy
- add `/health`
- add Dockerfile / docker-compose / `.env.example`

### Migration impact
- the repo begins transitioning toward the final Go-only shape
- the old app remains present, but the new app foundation now exists

### Acceptance criteria
- Go app boots
- router works
- Mongo connects
- `/health` works
- root-level Go structure is established

---

## Milestone 2 — Auth and Protected Shell

### Goal
Move authentication into the Go app.

### Tasks
- register/login/logout
- password hashing
- session persistence
- HTTP-only session cookie
- protected route middleware
- localized login/register pages
- protected shell layout
- locale-aware redirects

### Migration impact
- auth ownership moves from old implementation to Go

### Acceptance criteria
- user can register and log in through the Go app
- protected routes work
- logout works
- locale-aware auth flow works

---

## Milestone 3 — Core Library CRUD

### Goal
Implement the basic library experience in the Go app.

### Tasks
- dashboard
- catalog
- wishlist
- detail page
- manual entry creation
- manual entry update/delete
- separate My copy / Media details display
- repositories for `media_records` and `library_entries`

### Migration impact
- the Go app becomes usable for core personal library management even before provider flows are ported

### Acceptance criteria
- catalog/wishlist/detail work in Go
- manual CRUD works
- forms work without JS

---

## Milestone 4 — Server-rendered Search

### Goal
Implement provider search in the Go app.

### Tasks
- search page
- text query + media type form
- provider adapter wiring
- normalized search result rendering
- add-to-catalog / add-to-wishlist entry points

### Migration impact
- search behavior now lives in Go
- old search implementation becomes reference only

### Acceptance criteria
- search works in Go
- results render clearly
- baseline search works without JS

---

## Milestone 5 — Import and Dedupe

### Goal
Port canonical import and dedupe logic into the Go app.

### Tasks
- provider-ref import
- normalized detail fetch
- exact provider-ref dedupe
- heuristic fallback dedupe
- create/reuse `media_record`
- optional linked `library_entry`
- duplicate-entry handling
- refresh scaffold

### Migration impact
- canonical media import now belongs to the Go app
- old import logic is no longer strategically important

### Acceptance criteria
- import works
- dedupe works safely
- search-to-import-to-library flow works

---

## Milestone 6 — htmx Enhancements

### Goal
Add selective progressive enhancement to the Go app.

### Tasks
- search result refresh
- filter refresh
- pagination updates
- inline notices
- add-to-library feedback fragments

### Migration impact
- Go app UX improves without changing the HTML-first architecture

### Acceptance criteria
- selected pages feel more responsive
- non-JS fallback still works

---

## Milestone 7 — Barcode Lookup Backend

### Goal
Implement barcode lookup orchestration in the Go app.

### Tasks
- barcode normalization
- local-first matches
- ordered provider stages
- normalized candidate response
- explicit fallback object
- optional scan log persistence

### Migration impact
- barcode lookup now belongs to the Go app backend

### Acceptance criteria
- books and albums work best
- fallback guidance is explicit
- no auto-save on lookup

---

## Milestone 8 — Scan Page with Small JS

### Goal
Implement the scan page in the Go app.

### Tasks
- scan page shell
- `scan.js`
- camera lifecycle
- duplicate-read prevention
- barcode lookup integration
- candidate rendering
- add-to-catalog / add-to-wishlist actions
- manual fallback

### Migration impact
- scanner UX now belongs to the Go app
- old scan page is obsolete after this stabilizes

### Acceptance criteria
- scan flow works
- manual fallback works
- explicit save action works

---

## Milestone 9 — Localization Polish

### Goal
Bring English/Japanese parity to the Go app.

### Tasks
- translation files
- server-side translation lookup
- locale switcher
- localized templates
- locale-preserving redirects/navigation

### Migration impact
- Go app reaches language parity with the intended v1 feature set

### Acceptance criteria
- `/en/...` and `/ja/...` work
- strings are localized across current pages

---

## Milestone 10 — Provider Reliability

### Goal
Port provider reliability improvements into the Go app.

### Tasks
- provider cache
- configurable TTLs
- in-flight dedupe
- structured provider errors
- MusicBrainz throttle/cooldown
- normalized fallback behavior

### Migration impact
- provider reliability now belongs to Go
- old reliability logic is no longer needed

### Acceptance criteria
- cache works
- repeated upstream calls are reduced
- partial provider failure does not break flows

---

## Milestone 11 — Remove the Old Implementation

### Goal
Intentionally remove the old Next.js/NestJS app once the Go app has replacement coverage.

### Tasks
- verify replacement coverage
- remove obsolete code from:
  - `apps/web`
  - `apps/api`
- remove obsolete JS/TS config
- update scripts and build flow
- clean repo structure around the Go app

### Migration impact
- repo transitions from temporary coexistence to final single-app structure

### Acceptance criteria
- old app structure is removed or nearly removed
- repo now clearly centers on the Go app

---

## Milestone 12 — Deployment, Docs, and Release Polish

### Goal
Finalize the repository as a Go-only application.

### Tasks
- finalize deployment docs
- finalize Docker setup
- finalize embed strategy
- update README
- add architecture docs
- add migration notes
- add “why Go” explanation
- add screenshots/placeholders
- align docs with actual implementation

### Migration impact
- rewrite is complete
- repo is ready for presentation and continued iteration

### Acceptance criteria
- repo reads as a single intentional Go app
- docs are clear
- deployment is documented
- old architecture is no longer the center of the repo

---

## Recommended Order

1. establish Go foundation  
2. auth  
3. core CRUD  
4. search  
5. import/dedupe  
6. htmx upgrades  
7. barcode backend  
8. scan page  
9. localization  
10. provider reliability  
11. remove old implementation  
12. deployment/docs/release polish