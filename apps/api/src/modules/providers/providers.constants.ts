export const PROVIDER_ADAPTERS = Symbol("PROVIDER_ADAPTERS");

export const PROVIDER_CACHE_TTLS = {
  searchMs: 12 * 60 * 60 * 1000,
  detailMs: 7 * 24 * 60 * 60 * 1000,
  barcodeMs: 7 * 24 * 60 * 60 * 1000
} as const;
