# Provider Notes

## Purpose

This project uses providers for metadata lookup, not as a full mirrored data warehouse. The API normalizes provider responses into internal record shapes and stores only the fields it needs.

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
- the API includes MusicBrainz-specific throttling settings through env vars
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

- provider access belongs to `apps/api`
- provider contracts and normalization helpers belong to `packages/provider-sdk`
- cached responses are stored in `provider_cache`
- barcode lookup may return failures and fallback hints instead of pretending every provider can resolve every code

## Practical Portfolio Value

These integrations help demonstrate:

- adapter-based backend design
- rate-limit-aware API work
- metadata normalization
- clear frontend/backend responsibility boundaries
