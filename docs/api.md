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
| GET | `/{locale}/library/new` |
| GET | `/{locale}/library/{entryId}` |
| GET | `/{locale}/library/{entryId}/edit` |
| GET | `/{locale}/scan` |
| GET | `/{locale}/settings` |
| POST | `/{locale}/logout` |

## Actions (non-locale POST routes)

| Method | Path |
|--------|------|
| POST | `/library` |
| POST | `/library/attach` |
| POST | `/library/{entryId}/update` |
| POST | `/library/{entryId}/delete` |
| POST | `/media/import` |
| POST | `/media/refresh/{mediaRecordId}` |
| POST | `/barcode/lookup` |

Barcode lookup returns JSON for the scan page script; it does **not** create library entries without an explicit user action elsewhere.

## Static files

| Path | Purpose |
|------|---------|
| `/static/*` | CSS, JS (e.g. barcode scanning) |

## Auth

Sessions use an HTTP-only cookie (name from `SESSION_COOKIE_NAME`, default `mlm_session`). Configuration is documented in [`internal/config/config.go`](../internal/config/config.go).

## Related docs

- Route map: [`docs/routes.md`](routes.md)
- Router wiring: [`internal/app/app.go`](../internal/app/app.go)
