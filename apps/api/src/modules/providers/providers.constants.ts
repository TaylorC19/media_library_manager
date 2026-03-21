export const PROVIDER_ADAPTERS = Symbol("PROVIDER_ADAPTERS");

export const PROVIDER_CACHE_VERSION = "v2";

export const PROVIDER_CACHE_TTLS = {
  searchMs: 60 * 60 * 1000,
  negativeSearchMs: 5 * 60 * 1000,
  detailMs: 7 * 24 * 60 * 60 * 1000,
  negativeDetailMs: 15 * 60 * 1000,
  barcodeMs: 7 * 24 * 60 * 60 * 1000,
  negativeBarcodeMs: 15 * 60 * 1000
} as const;
