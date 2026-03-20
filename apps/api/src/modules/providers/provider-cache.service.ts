import { Injectable } from "@nestjs/common";
import type { MediaType, ProviderName } from "@media-library/types";
import { normalizeBarcode } from "@media-library/provider-sdk";
import { PROVIDER_CACHE_TTLS } from "./providers.constants";
import { ProviderCacheRepository } from "./repositories/provider-cache.repository";

@Injectable()
export class ProviderCacheService {
  constructor(
    private readonly providerCacheRepository: ProviderCacheRepository
  ) {}

  getSearchCacheKey(mediaType: MediaType, query: string): string {
    return `search:${mediaType}:${query.trim().toLowerCase()}`;
  }

  getDetailsCacheKey(mediaType: MediaType, providerId: string): string {
    return `detail:${mediaType}:${providerId}`;
  }

  getBarcodeCacheKey(mediaType: MediaType | undefined, barcode: string): string {
    return `barcode:${mediaType ?? "any"}:${normalizeBarcode(barcode) ?? barcode}`;
  }

  async get<Payload>(
    provider: ProviderName,
    cacheKey: string
  ): Promise<{ hit: boolean; value: Payload | null }> {
    const cacheEntry = await this.providerCacheRepository.findActive<Payload>(
      provider,
      cacheKey
    );

    if (!cacheEntry) {
      return {
        hit: false,
        value: null
      };
    }

    return {
      hit: true,
      value: cacheEntry.payload
    };
  }

  async set<Payload>(
    provider: ProviderName,
    cacheKey: string,
    payload: Payload,
    ttlMs: number
  ): Promise<void> {
    await this.providerCacheRepository.set(
      provider,
      cacheKey,
      payload,
      new Date(Date.now() + ttlMs)
    );
  }

  async wrap<Payload>(options: {
    provider: ProviderName;
    cacheKey: string;
    ttlMs: number;
    loader: () => Promise<Payload>;
  }): Promise<Payload> {
    const cachedValue = await this.get<Payload>(
      options.provider,
      options.cacheKey
    );
    if (cachedValue.hit) {
      return cachedValue.value as Payload;
    }

    const value = await options.loader();
    await this.set(options.provider, options.cacheKey, value, options.ttlMs);
    return value;
  }

  getSearchTtlMs(): number {
    return PROVIDER_CACHE_TTLS.searchMs;
  }

  getDetailsTtlMs(): number {
    return PROVIDER_CACHE_TTLS.detailMs;
  }

  getBarcodeTtlMs(): number {
    return PROVIDER_CACHE_TTLS.barcodeMs;
  }
}
