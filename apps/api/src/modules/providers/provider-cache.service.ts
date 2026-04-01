import { Injectable } from "@nestjs/common";
import type { MediaType, ProviderName } from "@media-library/types";
import { normalizeBarcode } from "@media-library/provider-sdk";
import { ProviderReliabilityService } from "./provider-reliability.service";
import { ProviderCacheRepository } from "./repositories/provider-cache.repository";
import type { ProviderCacheOperation } from "./providers.types";

@Injectable()
export class ProviderCacheService {
  private readonly inFlightRequests = new Map<string, Promise<unknown>>();

  constructor(
    private readonly providerCacheRepository: ProviderCacheRepository,
    private readonly providerReliabilityService: ProviderReliabilityService
  ) {}

  getSearchCacheKey(mediaType: MediaType, query: string): string {
    return this.withVersion(`search:${mediaType}:${query.trim().toLowerCase()}`);
  }

  getDetailsCacheKey(mediaType: MediaType, providerId: string): string {
    return this.withVersion(`detail:${mediaType}:${providerId}`);
  }

  getBarcodeCacheKey(mediaType: MediaType | undefined, barcode: string): string {
    return this.withVersion(
      `barcode:${mediaType ?? "any"}:${normalizeBarcode(barcode) ?? barcode}`
    );
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
    operation: ProviderCacheOperation;
    cacheKey: string;
    loader: () => Promise<Payload>;
  }): Promise<Payload> {
    const cachedValue = await this.get<Payload>(
      options.provider,
      options.cacheKey
    );
    if (cachedValue.hit) {
      return cachedValue.value as Payload;
    }

    const inFlightKey = `${options.provider}:${options.cacheKey}`;
    const existingRequest = this.inFlightRequests.get(inFlightKey) as
      | Promise<Payload>
      | undefined;

    if (existingRequest) {
      return existingRequest;
    }

    const request = this.loadAndCache(options);
    this.inFlightRequests.set(inFlightKey, request);

    try {
      return await request;
    } finally {
      this.inFlightRequests.delete(inFlightKey);
    }
  }

  private withVersion(cacheKey: string): string {
    return `${this.providerReliabilityService.getCacheVersionPrefix()}:${cacheKey}`;
  }

  private async loadAndCache<Payload>(options: {
    provider: ProviderName;
    operation: ProviderCacheOperation;
    cacheKey: string;
    loader: () => Promise<Payload>;
  }): Promise<Payload> {
    const policy = this.providerReliabilityService.getCachePolicy(options.operation);
    const value = await options.loader();
    const ttlMs = this.providerReliabilityService.isNegativeCacheEligible(
      options.operation,
      value
    )
      ? policy.negativeTtlMs
      : policy.ttlMs;

    await this.set(options.provider, options.cacheKey, value, ttlMs);

    return value;
  }
}
