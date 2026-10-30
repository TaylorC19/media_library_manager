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
| POST | `/{locale}/logout` | Logout |

### Search query parameters

Search is **GET only**. Relevant query keys:

- `q` — search text  
- `media_type` — `movie` \| `tv` \| `album` \| `book` \| `game`  
- `page` — page number for in-memory pagination  
- `page_size` — optional page size  
- `mode=barcode` — opens the barcode section without running a title search.

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
