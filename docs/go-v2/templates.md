# Media Library Manager — Go Rewrite Route Map

## Overview

This document defines the route plan for the Go rewrite.

The route model should preserve the product behavior of the old application while moving ownership into the new Go app.

The old app in `apps/web` and `apps/api` is reference only.

The final route model should belong entirely to the Go application.

---

## Route Principles

- page routes are locale-prefixed
- the app is private/login-first
- page rendering is server-side
- action endpoints may be stable non-locale routes if helpful
- redirects must preserve locale where appropriate
- route design should support eventual removal of the old app

---

## Root Route

### `/`
- redirect to default or detected locale

---

## Public Page Routes

### Login
- `GET /{locale}/login`
- `POST /{locale}/login`

### Register
- `GET /{locale}/register`
- `POST /{locale}/register`

---

## Protected Page Routes

### Dashboard
- `GET /{locale}/`

### Catalog
- `GET /{locale}/catalog`

### Wishlist
- `GET /{locale}/wishlist`

### Search
- `GET /{locale}/search`
- `POST /{locale}/search`

### Library Detail
- `GET /{locale}/library/{entryId}`

### Scan
- `GET /{locale}/scan`

### Settings
- `GET /{locale}/settings`

### Logout
- `POST /{locale}/logout`

---

## Action Endpoints

These may remain stable non-locale routes if that keeps the server-rendered form flow simpler.

### Library actions
- `POST /library`
- `POST /library/{entryId}/update`
- `POST /library/{entryId}/delete`

### Media actions
- `POST /media/import`
- `POST /media/refresh/{mediaRecordId}`

### Barcode actions
- `POST /barcode/lookup`

---

## htmx / Fragment Behavior

Preferred approach:
- same page routes can return full page or fragment depending on request context

Optional later:
- dedicated fragment routes only if they improve clarity

Examples:
- search results section
- library list section
- pagination section
- inline notices

---

## Redirect Rules

### Unauthenticated protected access
Redirect to:
- `/{locale}/login`

### Successful login
Redirect to:
- `/{locale}/`

### Successful register
Redirect to:
- `/{locale}/`

### Successful add/import/save
Redirect to:
- `/{locale}/library/{entryId}`

### Locale switch
Preserve current path when possible

---

## Migration Notes

### During migration
- old route behavior may be referenced for parity
- new Go handlers should own route behavior going forward
- avoid creating new product behavior inside old route layers

### Final state
All production route ownership should belong to the Go app only.

---

## v1-to-Go Mapping Intention

### Keep conceptually
- locale-prefixed page structure
- protected/private app
- catalog/wishlist/search/detail/scan/settings routes

### Change technically
- server-rendered Go routes instead of Next.js route tree
- Go handlers instead of Nest/Next route ownership
- HTML-first responses instead of SPA-like client ownership

---

## Handler Ownership

### Auth
- login
- register
- logout

### Library
- dashboard
- catalog
- wishlist
- detail
- create/update/delete entry

### Search
- search page
- search results/fragments

### Media
- import
- refresh

### Barcode
- lookup

### Settings
- settings page

---

## Success Criteria

The route migration is successful when:
- the Go app owns all user-facing route behavior
- the old route ownership is no longer needed
- locale-aware navigation works
- HTML-first rendering works across the app

# Media Library Manager — Go Rewrite Template Architecture

## Overview

This document defines the template strategy for the Go rewrite.

The template system should support:

- server-rendered HTML
- reusable partials
- localized UI strings
- optional htmx fragments
- a clean HTML-first architecture

The template layer should be built for the final Go app, not for long-term coexistence with the old frontend.

---

## Template Principles

- templates render HTML, not business logic
- templates receive structured view models
- templates should remain readable and explicit
- templates should support localization cleanly
- templates should support both full-page and enhanced interactions
- templates should feel native to a Go/HTML-first app

## Interaction Principles

The app is not restricted to plain form posts only.

Templates may support:
- normal HTML form submission
- htmx-enhanced submission
- JS-enhanced submission using fetch or similar browser APIs

Choose the interaction style that best fits the page, while keeping these rules:

- the backend remains the source of truth
- templates do not become a client-side app runtime
- business logic does not move into browser code
- interactions should stay readable and maintainable

## HTML-first Rules

- normal links and forms should remain easy to support
- htmx is enhancement, not the core architecture
- JavaScript is allowed where it meaningfully improves UX
- templates must not assume SPA-style global client state
- scanner behavior is the clearest case for browser-managed JS

## htmx Guidelines

Use htmx selectively when server-rendered partial HTML improves UX.

### Good use cases
- search result refresh
- library list updates
- pagination
- inline notices
- add/remove feedback
- create/edit flows when partial HTML responses are cleaner than full-page redirects

### Vanilla JS use cases
- camera lifecycle
- barcode scanning
- async form submission where that is clearer than htmx
- richer UI interactions that remain page-scoped

### Avoid
- complex SPA-style state orchestration
- moving canonical data ownership into the client
- rebuilding React-style component behavior in ad hoc JS

## Form Rules

- semantic form controls
- field-level validation display
- preserve submitted values on validation failure where appropriate
- localized labels/buttons
- explicit submit actions
- forms may be:
  - normal HTML
  - htmx-enhanced
  - JS-enhanced
- choose the simplest interaction model that gives a good user experience

---

## Migration Principle

The template system is part of the replacement architecture.

It should not try to recreate React components or client-side state patterns from the old app.

Use the old app only as reference for:
- page content
- information hierarchy
- UX expectations

Do not translate the old component model directly into templates.

---

## Suggested Template Structure

```txt
internal/views/templates/
  layout/
    base.html
    app_shell.html
    auth_shell.html

  partials/
    nav.html
    flash.html
    language_switcher.html
    page_header.html
    empty_state.html
    form_errors.html
    search_results.html
    search_result_card.html
    library_entry_list.html
    library_entry_card.html
    pagination.html
    scan_status.html
    scan_candidates.html

  pages/
    login.html
    register.html
    dashboard.html
    catalog.html
    wishlist.html
    search.html
    library_detail.html
    scan.html
    settings.html




---

## `docs/go-v2/data-model.md`

```md
# Media Library Manager — Go Rewrite Data Model

## Overview

This document defines the data model for the Go rewrite.

The rewrite preserves the core domain model from the current implementation while moving ownership into the Go app.

The repository currently contains an older implementation in:

- `apps/web`
- `apps/api`

These are the reference implementation only during migration.

The final Go app should own the data model and persistence behavior completely.

---

## Data Model Principles

- MongoDB remains the persistence layer
- canonical metadata belongs in `media_records`
- user-owned state belongs in `library_entries`
- sessions are server-side
- provider metadata is normalized before storage
- raw provider payloads are not the application schema
- safe dedupe is preferred over aggressive merging

---

## Migration Principle

The data model itself is largely preserved from the old implementation.

What changes is:
- implementation language
- repository/service structure
- route ownership
- rendering model

Do not redesign the domain unnecessarily during the rewrite unless it clearly improves correctness or simplicity.

---

## Collections

### Required
- `users`
- `sessions`
- `media_records`
- `library_entries`

### Optional
- `provider_cache`
- `scan_logs`

---

## `users`

### Purpose
Stores user identity and app-level settings.

### Fields
- `_id`
- `username`
- `passwordHash`
- `displayName`
- `settings`
- `createdAt`
- `updatedAt`

### Rules
- `username` must be unique
- no email required in initial rewrite scope
- password hashes only

---

## `sessions`

### Purpose
Stores server-side session state.

### Fields
- `_id`
- `userId`
- `tokenHash`
- `expiresAt`
- `createdAt`
- `lastUsedAt`
- optional `userAgent`
- optional `ipAddress`

### Rules
- store hashed token only
- cookie holds raw session token
- session resolved server-side
- delete session on logout

---

## `media_records`

### Purpose
Stores canonical normalized media metadata.

### Fields
- `_id`
- `source`
- `mediaType`
- `title`
- `sortTitle`
- `releaseDate`
- `year`
- `imageUrl`
- `summary`
- `providerRefs`
- `externalRatings`
- `barcodeCandidates`
- `details`
- `lastSyncedAt`
- `createdAt`
- `updatedAt`

### `source`
Allowed values:
- `manual`
- `provider`

### `mediaType`
Allowed values:
- `movie`
- `tv`
- `album`
- `book`
- `game`

### Rules
- stores canonical/provider-owned metadata
- does not store notes/tags/purchase date/bucket/format
- does not become a raw provider dump

---

## `library_entries`

### Purpose
Stores the user’s relationship to a media record.

### Fields
- `_id`
- `userId`
- `mediaRecordId`
- `bucket`
- `mediaType`
- `format`
- `barcode`
- `purchaseDate`
- `notes`
- `tags`
- `createdAt`
- `updatedAt`

### `bucket`
Allowed values:
- `catalog`
- `wishlist`

### Rules
- belongs to one user
- points to one media record
- stores user-owned state only
- may copy barcode from actual scan/manual input
- must not become a duplicate metadata store

---

## `provider_cache`

### Purpose
Stores cached provider responses.

### Fields
- `_id`
- `provider`
- `cacheKey`
- `payload`
- `expiresAt`
- `createdAt`

### Rules
- cache search/detail/barcode provider responses
- TTL behavior configurable
- not canonical user-facing data

---

## `scan_logs`

### Purpose
Stores barcode lookup history for an authenticated user.

### Fields
- `_id`
- `userId`
- `barcode`
- `matchedMediaType`
- `matchedProvider`
- `createdAt`

### Rules
- user-scoped
- optional/supportive
- not required for core product correctness

---

## Domain Relationships

### User → LibraryEntry
One user has many library entries.

### LibraryEntry → MediaRecord
Many entries may point to one canonical media record.

### User → Session
One user may have multiple sessions.

### MediaRecord → ProviderRefs
One media record may store one or more provider references.

---

## Dedupe Rules

### Media import dedupe order
1. exact provider-ref match
2. exact ISBN / barcode candidate match where subtype supports it
3. normalized title + year + primary creator heuristic
4. otherwise create a new media record

### Safety rule
If the match is ambiguous, create a new media record instead of silently merging.

---

## Import Rules

### Manual creation
Manual creation still creates a `media_record`, but:
- `source = manual`
- sparse details allowed
- provider refs may be empty

### Provider import
Provider import:
- creates or reuses canonical `media_record`
- uses `source = provider`
- preserves provider refs
- stamps sync timestamps

---

## Refresh Rules

Refreshing a media record may update provider-owned metadata only.

Refreshing must not update:
- bucket
- format
- notes
- tags
- purchase date
- user-owned barcode in `library_entries`

---

## Search Rules

Provider search results are transient display objects.

They are not canonical stored records until import happens.

---

## Barcode Rules

Barcode lookup is lookup only.

A barcode scan may:
- return candidates
- return local matches
- help prefill manual search

A barcode scan must not:
- auto-import media
- auto-create entries
- auto-save without user action

---

## Indexing Guidance

### users
- unique username

### sessions
- tokenHash
- expiresAt

### library_entries
- userId + bucket + mediaType
- userId + createdAt
- userId + mediaRecordId + bucket + format when useful for duplicate prevention

### media_records
- providerRefs.*
- mediaType + title + year
- barcodeCandidates

### provider_cache
- provider + cacheKey
- expiresAt

### scan_logs
- userId + createdAt
- barcode

---

## Rewrite Success Criteria

The data model migration is successful when:
- the Go app fully owns persistence behavior
- canonical metadata and user-owned state remain separate
- imports remain safe and dedupe-aware
- the old app is no longer needed to reason about data behavior

