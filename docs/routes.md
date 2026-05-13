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
| GET | `/{locale}/` | Dashboard |
| GET | `/{locale}/catalog` | Catalog list (filters/pagination may use htmx). |
| GET | `/{locale}/wishlist` | Wishlist list |
| GET | `/{locale}/search` | Search UI and results |
| GET | `/{locale}/library/new` | Create manual entry form |
| GET | `/{locale}/library/{entryId}` | Detail |
| GET | `/{locale}/library/{entryId}/edit` | Edit entry form |
| GET | `/{locale}/scan` | Barcode scan page (JS-enhanced) |
| GET | `/{locale}/settings` | Settings |
| POST | `/{locale}/logout` | Logout |

### Search query parameters

Search is **GET only**. Relevant query keys:

- `q` — search text  
- `media_type` — `movie` \| `tv` \| `album` \| `book` \| `game`  
- `page` — page number for in-memory pagination  
- `page_size` — optional page size  

When the request is an **htmx** request (`HX-Request: true`), the handler returns the **`partials/search_results`** template fragment instead of the full `pages/search.html` shell so the results region can swap in place.

## Actions (POST, mostly non-locale)

| Method | Path |
|--------|------|
| POST | `/library` | Create library entry (e.g. manual create). |
| POST | `/library/attach` | Attach entry to existing media (flows that post here). |
| POST | `/library/{entryId}/update` | Update entry |
| POST | `/library/{entryId}/delete` | Delete entry |
| POST | `/media/import` | Import from provider ref into catalog/wishlist |
| POST | `/media/refresh/{mediaRecordId}` | Refresh canonical metadata (htmx may target inline notice). |
| POST | `/barcode/lookup` | Barcode lookup (JSON for scan page; does **not** auto-save). |

## htmx vs full page (summary)

| Area | Pattern |
|------|---------|
| Search | Same `GET /{locale}/search`; fragment vs full page from handler. |
| Library lists / pagination | `hx-get` on filter/pagination controls; targets `#library-list-section` or similar. |
| Import / refresh | Forms may use `hx-post` with a small inline notice target. |

Exact attributes live in templates under [`internal/views/templates/`](../internal/views/templates/).

## Redirects (intended behavior)

- Successful auth: back to `/{locale}/`  
- Unauthenticated protected access: `/{locale}/login`  
- After save/import: typically `/{locale}/library/{entryId}` (see handlers)  

Locale switching preserves path where the language switcher replaces the first path segment.
