# Route map

This document describes **page** vs **action** routes and how **htmx** affects responses. The source of truth for wiring is [`internal/app/app.go`](../internal/app/app.go). A compact HTTP table lives in [`docs/api.md`](api.md).

## Conventions

- **`{locale}`** — `en` or `ja`.
- **Protected routes** require a valid session cookie (see [`docs/api.md`](api.md)).
- **Action routes** below `/library`, `/media`, and `/barcode` omit the locale segment; handlers redirect back to locale-prefixed pages after work completes.

## Root

| Method | Path | Behavior |
|--------|------|----------|
| GET | `/` | Redirect to `/{defaultLocale}/` (from `DEFAULT_LOCALE`). |
| GET | `/health` | JSON health check (Docker / probes). |
| GET | `/static/*` | Static files (CSS, JS, vendored htmx, scan helpers). |

## Public (unauthenticated) pages

| Method | Path |
|--------|------|
| GET, POST | `/{locale}/login` |
| GET, POST | `/{locale}/register` |

Unauthenticated visitors hitting protected URLs are redirected to login (locale preserved via middleware).

## Protected pages

| Method | Path | Notes |
|--------|------|--------|
| GET | `/{locale}/` | My Library, Collection view (default landing page) |
| GET | `/{locale}/catalog` | Compatibility URL for My Library’s Collection view. |
| GET | `/{locale}/wishlist` | My Library, Wishlist view |
| GET | `/{locale}/search` | Title search, barcode entry, and camera scanning. Results open a details dialog with both save actions. |
| GET | `/{locale}/search/details` | Read-only metadata preview; `fragment=1` returns dialog content, otherwise a full page. |
| GET | `/{locale}/library/new` | Manual entry fallback from Search; prefills title query, media type, and barcode when supplied. |
| GET | `/{locale}/library/{entryId}` | Detail |
| GET | `/{locale}/library/{entryId}/edit` | Edit entry form |
| GET | `/{locale}/scan` | Compatibility redirect to Search with `mode=barcode`; preserves query parameters. |
| GET | `/{locale}/settings` | Settings |
| GET | `/{locale}/admin/users` | Admin-only account list/search with pagination; optional `q` and `role=admin|general|demo` filters combine. Empty role means all accounts, and General includes legacy accounts without a role. |
| POST | `/{locale}/admin/users` | Admin-only account creation; preserves the administrator’s session. |
| POST | `/{locale}/admin/users/{userId}/role` | Admin-only role change; self-demotion prohibited. |
| GET | `/{locale}/admin/users/{userId}/keys` | Admin-only personal-key status for TMDB, RAWG, and Discogs; never exposes values. |
| GET | `/{locale}/admin/users/{userId}/keys/clear` | Confirmation page for the selected `provider`; does not modify keys. |
| POST | `/{locale}/admin/users/{userId}/keys/clear` | Admin-only clearing of one personal override; requires `provider` and `confirm=clear`, checks current admin access, and rejects cross-origin submissions. |
| POST | `/{locale}/logout` | Logout |

### Search query parameters

Search is **GET only**. Relevant query keys:

- `q` — search text  
- `media_type` — optional: `movie` \| `tv` \| `album` \| `book` \| `game`; empty/omitted means All media.
- `page` — page number for in-memory pagination  
- `page_size` — optional page size  
- `mode=barcode` — opens the barcode section without running a title search.
- `view` — `grid` (default for Search) or `list`; retained through form submissions, See all links, and pagination.

With All media selected, lookups run concurrently for all five types. Each non-empty group shows up to three results and a See all link to the same query filtered by that type, starting on page 1. Empty groups are omitted; provider warnings remain scoped to their type. Type-filtered searches retain normal pagination.

When the request is an **htmx** request (`HX-Request: true`), the handler returns the **`partials/search_results`** template fragment instead of the full `pages/search.html` shell so the results region can swap in place.

### Library layout

My Library accepts `view=list` (default) or `view=grid`. Cover images appear in both layouts, with a placeholder when unavailable. Layout links preserve the current filters and page; filtering and pagination preserve `view`. Collection/Wishlist links retain the layout.

### Search detail previews

`GET /{locale}/search/details` accepts `provider`, `external_id`, `media_type`, and `tmdb_kind` for provider results, or `media_record_id` for shared local metadata. It fetches full metadata without importing records or creating entries. Each preview offers separate POST actions for Collection and Wishlist. The native dialog supports Escape, close, focus restoration, loading errors, and retry. Links navigate to a full detail page when JavaScript is unavailable. Barcode-only sources without an import adapter offer manual creation.

## Actions (POST, mostly non-locale)

| Method | Path |
|--------|------|
| POST | `/library` | Create library entry (e.g. manual create). |
| POST | `/library/attach` | Attach entry to existing media (flows that post here). |
| POST | `/library/{entryId}/update` | Update entry |
| POST | `/library/{entryId}/delete` | Delete entry |
| POST | `/media/import` | Import from provider ref into catalog/wishlist |
| POST | `/media/refresh/{mediaRecordId}` | Refresh canonical metadata (htmx may target inline notice). |
| POST | `/barcode/lookup` | Barcode lookup (JSON for the barcode section in Search; does **not** auto-save). |
| POST | `/{locale}/settings/providers/test` | Authenticated, read-only credential check for one provider; never saves settings. |
| POST | `/{locale}/settings/providers` | Save/remove only the signed-in account’s personal provider keys; blank fields preserve values. |
| POST | `/settings/backup/export` | Download full-database backup (gzip JSON). |
| POST | `/settings/backup/import` | Restore full database from backup file (requires confirmation). |

See [`docs/backup.md`](backup.md).

## htmx vs full page (summary)

| Area | Pattern |
|------|---------|
| Search | Standard GET form and pagination use only lookup criteria; the handler also supports fragment responses for `HX-Request`. |
| Library lists / pagination | `hx-get` on filter/pagination controls; targets `#library-list-section` or similar. |
| Import / refresh | Forms may use `hx-post` with a small inline notice target. |

Exact attributes live in templates under [`internal/views/templates/`](../internal/views/templates/).

## Redirects (intended behavior)

- Successful auth: back to `/{locale}/`  
- Unauthenticated protected access: `/{locale}/login`  
- After save/import: typically `/{locale}/library/{entryId}` (see handlers)  

Locale switching preserves path where the language switcher replaces the first path segment.

## Role permissions

All protected routes require authentication. Full-database export/restore additionally require an admin role and reject cross-origin browser submissions. Library creation/attachment/editing/deletion, provider imports, metadata refresh, and provider-key saving require a writable account (admin or general); demo accounts receive HTTP 403. New/edit form pages are restricted in the same way. Public registration always creates general users; admins can create accounts and change other accounts’ roles in User management. Self-demotion is blocked, changes are serialized in the single-process server, and the actor’s current role is checked again before each change. The server-side `cmd/users` command remains available for initial promotion and recovery. Accounts without a stored role are general.

Pagination clamps an out-of-range page to the same last page for both data slicing and footer labels. Provider-import forms retain scanned barcodes on the user's copy, and local lookup checks that account's saved barcodes. Refresh verifies ownership, fetches uncached provider details, replaces cached details after success, and updates the stored metadata. Deletion returns to the entry's original bucket. Language-switch links are updated from the live URL after HTMX filtering/pagination and history navigation.
