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
| GET | `/{locale}/search/group` | Authenticated HTML fragment for one independent media-type preview; accepts `q`, `media_type`, `view`, and `page_size`. |
| GET | `/{locale}/search/details` | Read-only metadata preview; `fragment=1` returns dialog content, otherwise a full page. |
| GET | `/{locale}/library/new` | Manual entry fallback from Search; prefills title query, media type, and barcode when supplied. |
| GET | `/{locale}/library/{entryId}` | Detail |
| GET | `/{locale}/library/{entryId}/delete` | Read-only confirmation naming the owned item and its copy details; writable accounts only. |
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
- `page` — page number for provider-backed pagination
- `page_size` — optional page size  
- `mode=barcode` — opens the barcode section without running a title search.
- `view` — `grid` (default for Search) or `list`; retained through form submissions, See all links, and pagination.
- `complete=1` — synchronous All media fallback for browsers without JavaScript.

With All media selected, the page renders immediately with five loading placeholders. JavaScript requests each media type independently from `/{locale}/search/group`; each finished group appears in its original position without waiting for slower providers. Each non-empty group shows up to three results, personal saved badges, and a See all link to the query filtered by that type, starting on page 1. Empty groups disappear; the overall empty state appears after every group finishes successfully with no matches. Provider failures offer a retry for that group, while key-configuration warnings remain scoped to their type. Navigation cancels pending requests, and disconnected or superseded responses cannot update results. Browsers without JavaScript get a link to `complete=1`, which renders all groups synchronously. Type-filtered searches fetch the requested provider page instead of slicing an initial batch.

Title-search cards show **In Collection** and **In Wishlist** badges for the signed-in account in both List and Grid views, including All media previews and type-filtered pages. Both badges appear when appropriate. Status is checked in a batch for visible results by provider ID and media type (including TMDB movie/TV kind), never by title. Manual records without provider references cannot be matched. Personal status is added after provider caching; a failed status check leaves results visible with a warning. Search responses are private and not cached.

When the request is an **htmx** request (`HX-Request: true`), the handler returns the **`partials/search_results`** template fragment instead of the full `pages/search.html` shell so the results region can swap in place.

### Provider-backed pagination

Type-filtered search uses provider totals and requests the relevant page or offset. TMDB movie/TV results use native 20-item pages (up to the provider’s 500-page limit), translating app offsets and page sizes across native boundaries. Open Library and MusicBrainz use `offset`/`limit`; RAWG uses `page`/`page_size`. First-page metadata is cached and used to clamp out-of-range requests before fetching the requested page. Cache keys include page and page size under a new namespace, so old first-batch cache entries cannot mask later results. Grouped searches fetch only three preview results while retaining full accessible totals and See all links. Personal saved badges are added afterward for the visible results.

Pagination reserves Previous, centered page controls, and Next positions, with unavailable navigation and the current page rendered as disabled buttons. Narrow screens place page controls above the navigation row. Library pagination retains HTMX updates; search links retain the query, media type, layout, and page size through full navigation.

### Library layout

My Library accepts `view=list` (default) or `view=grid`. Cover images appear in both layouts, with a placeholder when unavailable. Layout links preserve the current filters and page; filtering and pagination preserve `view`. Collection/Wishlist links retain the layout.

### Remembered layouts and filters

Collection, Wishlist, and Search remember layout, media type, text query, and page size; Library views also remember format, tag, and sort order. Preferences are stored for up to one year in bounded, HttpOnly, SameSite=Lax browser cookies scoped by account ID and workspace. They work without JavaScript and do not modify account data, including for demo users. They are shared across languages within the same browser, and do not sync to other browsers/devices. Text filters longer than 256 bytes still work for the current request but are not persisted.

Missing query parameters inherit saved preferences; explicit values, including empty strings, take priority. Returning via a bare workspace URL restores preferences and starts at page 1. Page number, barcode mode, and synchronous fallback flags are not saved. Restored requests canonicalize their URL (or use `HX-Push-Url` for HTMX fragments), while generated pagination links encode empty filters explicitly. Clear filters clears saved filter fields and retains layout/page size and the Library sort order. Progressive search-group and detail requests do not update workspace preferences. Collection/Wishlist tabs continue carrying the chosen layout while restoring each bucket’s own filters.

### Library discovery

My Library supports `sort=recent` (default, entry creation date newest first), `title_asc`, `title_desc`, `year_desc`, and `year_asc`. Sorting happens before pagination, uses deterministic tie-breakers, and places unknown years last in either year direction. Title comparisons ignore case. Invalid sort values return HTTP 400.

`tag` matches one whole tag case-insensitively. The tag dropdown lists only tags from the signed-in account’s current Collection/Wishlist, merging case variants; the selected tag remains visible even if it no longer exists. Clicking a saved item’s tag opens that bucket filtered by the tag. Text query `q` matches titles, descriptions, notes, tags, and media-specific creator fields: authors, directors, TV creators, artists, and developers. Tag, text, media type, and format criteria combine.

Sort and tag choices persist with the existing browser preferences, across page links, layout changes, and locale switching. Existing preference cookies without these fields remain compatible. Clear removes the tag and other filters while keeping layout, sort, and page size. HTMX responses refresh the Clear link out of band so it retains the latest chosen sort. This feature retains in-memory filtering/sorting over matching entries; database-side Library pagination remains a separate scalability improvement.

### Moving saved items

Writable accounts see **Move to Collection** near the top of Wishlist item details. The POST changes only `bucket` and `updatedAt` on the owned Wishlist entry, preserving its ID, media link, notes, tags, format, barcode, purchase date, and creation date. It returns to that item with a success message. Repeating a completed move is harmless. If Collection already contains the same media record and format, the unique-entry constraint rejects the move: both entries remain unchanged and a message explains the conflict. Demo accounts cannot move items, and another account’s entry returns 404.

### Search detail previews

`GET /{locale}/search/details` accepts `provider`, `external_id`, `media_type`, and `tmdb_kind` for provider results, or `media_record_id` for shared local metadata. It fetches full metadata without importing records or creating entries. Personal saved copies are looked up separately from shared metadata and never cached in provider responses. Each preview offers Open saved item links for existing Collection/Wishlist copies (including their formats); unsaved buckets offer separate POST actions. The native dialog supports Escape, close, focus restoration, loading errors, and retry. Links navigate to a full detail page when JavaScript is unavailable. Barcode-only sources without an import adapter offer manual creation.

## Actions (POST, mostly non-locale)

| Method | Path |
|--------|------|
| POST | `/library` | Create library entry (e.g. manual create). |
| POST | `/library/attach` | Attach entry to existing media (flows that post here). |
| POST | `/library/{entryId}/update` | Update entry |
| POST | `/library/{entryId}/move-to-collection` | Move an owned Wishlist entry to Collection; preserves entry ID and all copy fields. Writable accounts only; rejects cross-origin submissions. |
| POST | `/library/{entryId}/delete` | Delete an owned entry only with `confirm=delete`; enforces writable access and rejects cross-origin submissions. |
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


### Saving from search dialogs

JavaScript intercepts the dialog’s import/attach forms and submits to the existing `/media/import` or `/library/attach` endpoint with `X-Search-Dialog: 1`. Success returns JSON containing only the saved `entryId`, `bucket`, a local `url`, and an `existing` flag. It does not redirect or set a success flash cookie. The dialog stays open, announces success, and replaces the saved bucket’s action with Open saved item. Matching visible result cards gain the corresponding badge immediately, preserving the search URL, page, layout, and scroll position.

Save buttons are disabled while a request is pending. Failed requests leave the form available for retry and never add a badge. Closing a dialog does not cancel an in-flight write: its acknowledgement can update the matching visible badge without reopening the dialog or overwriting a different item’s preview. A preview that was already loading when a save completed reconciles that acknowledgement before displaying its actions.

Both save endpoints enforce writable accounts and same-origin submissions. Normal forms retain the existing redirect-to-detail behavior when JavaScript is unavailable. Local barcode matches use the attach endpoint and participate in the same dialog behavior; barcode-only manual-create fallbacks remain ordinary navigation.


### Deletion confirmation

Delete on an item’s detail page opens `/{locale}/library/{entryId}/delete`. This read-only page shows its title, Collection/Wishlist location, media type, year, format, and barcode when available. Cancel returns to that same item. The final POST requires `confirm=delete`; missing or invalid confirmation re-renders the review page with HTTP 400 and leaves the item unchanged. Both the confirmation GET and deletion POST enforce writable access and ownership; the POST also checks origin. A confirmed deletion removes only the selected saved copy and returns to its original bucket. The page explains that deletion cannot be undone; Trash/Undo is not implemented.
