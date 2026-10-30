# HTTP surface (Go app)

The product is a **server-rendered** Go application. There is no separate public JSON API for third-party clients; the browser uses HTML pages, forms, and a small set of JSON endpoints used by the scan/barcode flows.

## Health

### `GET /health`

JSON body includes at least `ok`, `service`, and `database` name. Used for Docker and load balancer checks.

## Public pages

| Method | Path | Notes |
|--------|------|-------|
| GET | `/{locale}/login` | |
| POST | `/{locale}/login` | |
| GET | `/{locale}/register` | |
| POST | `/{locale}/register` | |

Locales are `en` or `ja`.

## Protected pages (session cookie required)

| Method | Path |
|--------|------|
| GET | `/{locale}/` |
| GET | `/{locale}/catalog` |
| GET | `/{locale}/wishlist` |
| GET | `/{locale}/search` |
| GET | `/{locale}/search/details` |
| GET | `/{locale}/search/group` |
| GET | `/{locale}/library/new` |
| GET | `/{locale}/library/{entryId}` |
| GET | `/{locale}/library/{entryId}/edit` |
| GET | `/{locale}/library/{entryId}/delete` |
| GET | `/{locale}/scan` |
| GET | `/{locale}/settings` |
| GET | `/{locale}/admin/users` |
| GET | `/{locale}/admin/users/{userId}/keys` |
| GET | `/{locale}/admin/users/{userId}/keys/clear` |
| POST | `/{locale}/logout` |

Admin-only account actions are `POST /{locale}/admin/users` (create) and `POST /{locale}/admin/users/{userId}/role` (change role). These routes require a current administrator, reject cross-origin submissions, and never return passwords or password hashes. Self-demotion is blocked.

Admins can inspect personal-key status and open a confirmation page with `?provider=tmdb|rawg|discogs`. `POST /{locale}/admin/users/{userId}/keys/clear` requires a provider and `confirm=clear`. It rechecks current administrator access, rejects cross-origin submissions, and clears only that account’s selected personal override. Key values are never returned; the server default applies afterward if configured. This does not revoke the key with the provider.

## Actions (non-locale POST routes)

| Method | Path |
|--------|------|
| POST | `/library` |
| POST | `/library/attach` |
| POST | `/library/{entryId}/update` |
| POST | `/library/{entryId}/move-to-collection` |
| POST | `/library/{entryId}/delete` |
| POST | `/media/import` |
| POST | `/media/refresh/{mediaRecordId}` |
| POST | `/barcode/lookup` |
| POST | `/settings/backup/export` |
| POST | `/settings/backup/import` |

The legacy `/{locale}/scan` page redirects to `/{locale}/search?mode=barcode`, preserving query parameters.

Barcode lookup returns JSON for the barcode section in Search; it does **not** create library entries without an explicit user action elsewhere.

Backup export returns `application/gzip` with a `.mlm-backup.json.gz` attachment. Import expects `multipart/form-data` with fields `backup` (file) and `confirm` (must equal `MONGODB_DATABASE`). See [`docs/backup.md`](backup.md).

## Static files

| Path | Purpose |
|------|---------|
| `/static/*` | CSS, JS (e.g. barcode scanning) |

## Auth

Sessions use an HTTP-only cookie (name from `SESSION_COOKIE_NAME`, default `mlm_session`). Configuration is documented in [`internal/config/config.go`](../internal/config/config.go).

## Related docs

- Route map: [`docs/routes.md`](routes.md)
- Router wiring: [`internal/app/app.go`](../internal/app/app.go)


Search-dialog saves to `/media/import` or `/library/attach` accept optional `format` and `notes`. Format must be relevant to the actual media type; an existing same-format copy is returned without replacing its notes. `X-Search-Dialog: 1` returns the existing saved-copy navigation JSON plus `format`. Ordinary form requests continue redirecting to the saved copy. One form uses Collection/Wishlist buttons named `bucket` to identify the destination.
