# API

## Overview

- Base URL in local development: `http://localhost:4000`
- No global API prefix is configured
- Protected routes require the session cookie
- CORS is enabled with credentials and uses `CORS_ORIGIN`

Most route payloads are shared through `packages/types`.

## Auth Model

The API uses session-cookie auth:

- login and register create a server-side session
- the browser receives an HTTP-only cookie
- protected routes read that cookie on subsequent requests
- `NODE_ENV=production` marks the cookie `Secure`

The session cookie name defaults to `mlib_session` and can be overridden with `SESSION_COOKIE_NAME`.

## Public Routes

### `GET /health`

Returns API and Mongo readiness information.

Example response:

```json
{
  "service": "api",
  "status": "ok",
  "timestamp": "2026-04-09T12:00:00.000Z",
  "uptimeSeconds": 184,
  "mongo": {
    "readyState": 1,
    "status": "connected"
  }
}
```

Notes:

- `status` is `ok` when Mongo is connected and `degraded` otherwise
- Docker health checks call this route directly

### `POST /auth/register`

Creates a user and starts a session.

Request body:

```json
{
  "username": "taylor",
  "password": "strong-password",
  "displayName": "Taylor"
}
```

Response:

```json
{
  "user": {
    "id": "user_id",
    "username": "taylor",
    "displayName": "Taylor"
  }
}
```

### `POST /auth/login`

Logs in an existing user and starts a session.

Request body:

```json
{
  "username": "taylor",
  "password": "strong-password"
}
```

Response:

```json
{
  "user": {
    "id": "user_id",
    "username": "taylor",
    "displayName": "Taylor"
  }
}
```

### `POST /auth/logout`

Clears the session cookie and removes the session when present.

Response:

```json
{
  "success": true
}
```

## Protected Routes

All routes below require a valid session cookie.

### Auth

#### `GET /auth/me`

Returns the current authenticated user.

### Library

#### `GET /library`

Lists the current user's library entries with joined media metadata.

Query parameters:

- `bucket`
- `mediaType`
- `tag`
- `search`
- `page`
- `pageSize`

Response shape:

```json
{
  "items": [
    {
      "entry": {
        "id": "entry_id",
        "bucket": "catalog",
        "mediaType": "album"
      },
      "media": {
        "id": "media_id",
        "mediaType": "album",
        "title": "Example Album"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "pageSize": 12,
    "total": 1,
    "totalPages": 1,
    "hasNextPage": false
  }
}
```

#### `POST /library`

Creates a library entry for the current user.

Key fields:

- `mediaRecordId`
- `bucket`
- optional `format`
- optional `barcode`
- optional `purchaseDate`
- optional `notes`
- optional `tags`

#### `GET /library/:entryId`

Returns one library entry plus its linked media record.

#### `PATCH /library/:entryId`

Updates user-owned fields on an existing library entry.

#### `DELETE /library/:entryId`

Deletes the user's library entry.

Response:

```json
{
  "success": true
}
```

### Media

#### `POST /media/manual`

Creates a minimal manual media record without calling a provider.

Used when the user wants to add an item directly.

#### `POST /media/import`

Imports a provider-backed media record into normalized storage.

Supported modes:

- `provider_ref`
- `search_result`

The request can optionally include an `entry` object to create or reuse a `library_entry` in the same operation.

Response fields:

- `mediaRecord`
- optional `libraryEntry`
- `wasExistingMediaRecord`
- optional `wasExistingLibraryEntry`

#### `GET /media/:mediaRecordId`

Returns a single media record.

#### `POST /media/refresh/:mediaRecordId`

Attempts to refresh an imported media record from a linked provider reference.

Response fields:

- `mediaRecord`
- `wasRefreshed`
- optional `refreshedFromProvider`
- optional `unavailableReason`

### Search

#### `GET /search`

Searches provider adapters for a single media type at a time.

Query parameters:

- `q`
- `mediaType`
- optional `limit`

The response includes:

- normalized `results`
- any provider `failures`
- the resolved `query`
- the requested `mediaType`

This route is protected, so provider-backed search happens only inside an authenticated session.

### Barcode

#### `POST /barcode/lookup`

Looks up a scanned barcode against local data and provider-backed candidates.

Request body:

```json
{
  "barcode": "012345678905",
  "preferredMediaType": "album"
}
```

Response fields:

- `barcode`
- `mediaType`
- `candidates`
- `failures`
- `fallback`

Candidate sources:

- `local`
- `provider`

Important behavior:

- the route returns candidate matches, not silent imports
- fallback metadata can point the UI toward manual search when barcode coverage is weak

## Error Shape

Most API errors use this JSON shape:

```json
{
  "message": "Human-readable error message",
  "code": "optional_machine_code"
}
```

The frontend currently relies primarily on the `message` field.
