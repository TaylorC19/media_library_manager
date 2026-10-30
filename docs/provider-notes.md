# Provider Notes

## Purpose

This project uses providers for metadata lookup, not as a full mirrored data warehouse. The Go server normalizes provider responses into internal record shapes and stores only the fields it needs.

## TMDB

Used for:

- `movie`
- `tv`

Notes:

- good default source for posters and mainstream movie/TV metadata
- provider references are stored so records can be refreshed later
- the frontend never calls TMDB directly

## MusicBrainz

Used for:

- `album`

Notes:

- this is the most rate-limit-sensitive provider in the current stack
- the Go server includes MusicBrainz-specific throttling settings through env vars
- response caching matters here more than for most other providers

Relevant env vars:

- `MUSICBRAINZ_USER_AGENT`
- `MUSICBRAINZ_MIN_INTERVAL_MS`
- `MUSICBRAINZ_RATE_LIMIT_COOLDOWN_MS`
- `MUSICBRAINZ_RETRYABLE_COOLDOWN_MS`

## Discogs

Used for:

- album enrichment
- some barcode-driven candidate matching

Notes:

- treated as a supplement rather than the core music source
- useful for release-level detail where available
- optional token-driven access through `DISCOGS_TOKEN`

## Open Library

Used for:

- `book`

Notes:

- a good fit for ISBN-oriented lookup
- supports the barcode-assisted book workflow well
- remains behind the same backend adapter boundary as the other providers

## RAWG

Used for:

- `game`

Notes:

- games are the most replaceable provider area in the project
- the adapter boundary is important here because game metadata sources may change over time
- current implementation keeps RAWG isolated behind the provider registry

## Implementation Rules

- provider access and adapters live in the Go application under `internal/providers` (and related services), not in the browser
- cached responses are stored in `provider_cache`
- barcode lookup may return failures and fallback hints instead of pretending every provider can resolve every code

## Practical Portfolio Value

These integrations help demonstrate:

- adapter-based backend design
- rate-limit-aware API work
- metadata normalization
- clear frontend/backend responsibility boundaries

## Configure API keys in Settings

Settings → Metadata providers lets each signed-in account configure personal **TMDB**, **RAWG**, and **Discogs** credentials. Changes apply to subsequent search, detail/import, and barcode requests without a restart. MusicBrainz and Open Library do not need API keys.

A personal key overrides the corresponding environment value for that account only. Without a personal key, `TMDB_API_KEY`, `RAWG_API_KEY`, and `DISCOGS_TOKEN` remain server defaults. Empty fields preserve saved values; Remove deletes only the personal override. Status labels indicate configuration. Each provider has a **Test key** button: it checks a candidate without saving, or checks the effective saved/server key if the input is blank. With Remove selected and no candidate entered, it tests the server default that would apply after saving. Tests bypass metadata caches and use read-only requests with a 10-second timeout. Responses distinguish accepted/rejected credentials, access restrictions, rate limits, and temporary failures; raw keys, provider payloads, and account identities are never returned. Metadata caching remains shared.

Keys are never populated into HTML or returned to the browser. They are stored as owner-readable/writable JSON files (0600) in an owner-only directory (0700), one file per account ID, under `PROVIDER_KEYS_DIR` (default `data/provider-keys`). These files contain unencrypted secrets, like an `.env` file; protect the host filesystem and use HTTPS for remote access. The directory is excluded from git and Docker build contexts.

User management → Manage saved keys lets administrators view personal-key presence and clear one provider override after a separate confirmation with a required checkbox. It never reveals key values. Clearing restores the server default for that account/provider if configured, preserves all other keys, and does not revoke credentials at the provider. Current admin access is checked again when clearing.

Database backups do **not** contain these files. Back up the directory separately using access controls suitable for secrets. Production Compose mounts a persistent `provider-keys` volume at `/home/app/data/provider-keys`; a standalone deployment must likewise preserve its configured directory across restarts/upgrades.

Credential probes use TMDB’s authenticated movie lookup, RAWG’s games listing with `page_size=1`, and Discogs’ authenticated identity lookup. Only a fixed success/failure message is rendered. Provider authentication references: [TMDB](https://developer.themoviedb.org/docs/authentication-application), [RAWG](https://api.rawg.io/docs/), [Discogs](https://www.discogs.com/developers/#page:user-identity,header:user-identity-identity).

MusicBrainz and Open Library also have **Test connection** buttons. They make small, uncached searches (`limit=1`) without API keys and verify the response structure. MusicBrainz uses the configured User-Agent and shares the app’s existing request throttle. These checks neither read nor write saved keys.
