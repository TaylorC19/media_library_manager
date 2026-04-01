export const PROVIDER_ADAPTERS = Symbol("PROVIDER_ADAPTERS");

export const DEFAULT_PROVIDER_CACHE_VERSION = "v3";

export const DEFAULT_PROVIDER_CACHE_TTLS = {
  searchMs: 12 * 60 * 60 * 1000,
  negativeSearchMs: 15 * 60 * 1000,
  detailMs: 7 * 24 * 60 * 60 * 1000,
  negativeDetailMs: 60 * 60 * 1000,
  barcodeMs: 7 * 24 * 60 * 60 * 1000,
  negativeBarcodeMs: 60 * 60 * 1000
} as const;

export const DEFAULT_MUSICBRAINZ_THROTTLE = {
  minIntervalMs: 1_100,
  rateLimitCooldownMs: 5_000,
  retryableCooldownMs: 2_000
} as const;
