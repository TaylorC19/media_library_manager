import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import {
  BaseMediaProvider,
  createProviderConfig,
  mapDiscogsDetails,
  mapDiscogsSearchResult,
  normalizeBarcode,
  normalizeMediaRecord,
  normalizeSearchResults,
  type DiscogsReleaseDetails,
  type DiscogsSearchResultItem,
  type ProviderCapabilities,
  type ProviderGetDetailsParams,
  type ProviderSearchByBarcodeParams,
  type ProviderSearchByTextParams
} from "@media-library/provider-sdk";
import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";
import { toProviderError } from "../errors/provider-error.utils";
import { ProviderCacheService } from "../provider-cache.service";
import { ProviderHttpService } from "../http/provider-http.service";

interface DiscogsSearchResponse {
  results?: DiscogsSearchResultItem[];
}

@Injectable()
export class DiscogsProvider extends BaseMediaProvider {
  readonly name: ProviderName = "discogs";
  readonly capabilities: ProviderCapabilities = {
    mediaTypes: ["album"],
    supportsTextSearch: true,
    supportsDetailLookup: true,
    supportsBarcodeSearch: true
  };

  constructor(
    private readonly configService: ConfigService,
    private readonly providerHttpService: ProviderHttpService,
    private readonly providerCacheService: ProviderCacheService
  ) {
    super();
  }

  get enabled(): boolean {
    return this.runtimeConfig.enabled;
  }

  async searchByText(
    params: ProviderSearchByTextParams
  ): Promise<NormalizedSearchResult[]> {
    if (!this.enabled || params.mediaType !== "album") {
      return [];
    }

    const cacheKey = this.providerCacheService.getSearchCacheKey(
      params.mediaType,
      params.query
    );

    try {
      return await this.providerCacheService.wrap({
        provider: this.name,
        cacheKey,
        ttlMs: this.providerCacheService.getSearchTtlMs(),
        loader: async () => {
          const response = await this.request<DiscogsSearchResponse>(
            "/database/search",
            {
              query: {
                type: "release",
                release_title: params.query,
                per_page: params.limit ?? 10
              }
            }
          );

          return normalizeSearchResults(
            (response.results ?? [])
              .map(mapDiscogsSearchResult)
              .filter((result): result is NormalizedSearchResult => result !== null)
          );
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "searchByText", error);
    }
  }

  async getDetails(
    params: ProviderGetDetailsParams
  ): Promise<NormalizedMediaRecordInput | null> {
    if (!this.enabled || params.mediaType !== "album") {
      return null;
    }

    const cacheKey = this.providerCacheService.getDetailsCacheKey(
      params.mediaType,
      params.providerId
    );

    try {
      return await this.providerCacheService.wrap({
        provider: this.name,
        cacheKey,
        ttlMs: this.providerCacheService.getDetailsTtlMs(),
        loader: async () => {
          const response = await this.request<DiscogsReleaseDetails>(
            `/releases/${params.providerId}`
          );

          return normalizeMediaRecord(mapDiscogsDetails(response));
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "getDetails", error);
    }
  }

  override async searchByBarcode(
    params: ProviderSearchByBarcodeParams
  ): Promise<NormalizedSearchResult[]> {
    if (!this.enabled) {
      return [];
    }

    if (params.mediaType !== undefined && params.mediaType !== "album") {
      return [];
    }

    const normalizedBarcode = normalizeBarcode(params.barcode);
    if (!normalizedBarcode) {
      return [];
    }

    const cacheKey = this.providerCacheService.getBarcodeCacheKey(
      params.mediaType,
      normalizedBarcode
    );

    try {
      return await this.providerCacheService.wrap({
        provider: this.name,
        cacheKey,
        ttlMs: this.providerCacheService.getBarcodeTtlMs(),
        loader: async () => {
          const response = await this.request<DiscogsSearchResponse>(
            "/database/search",
            {
              query: {
                type: "release",
                barcode: normalizedBarcode,
                per_page: params.limit ?? 10
              }
            }
          );

          return normalizeSearchResults(
            (response.results ?? [])
              .map(mapDiscogsSearchResult)
              .filter((result): result is NormalizedSearchResult => result !== null)
          );
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "searchByBarcode", error);
    }
  }

  private request<Response>(
    path: string,
    options?: Parameters<ProviderHttpService["getJson"]>[1]
  ): Promise<Response> {
    return this.providerHttpService.getJson<Response>(
      `${this.runtimeConfig.baseUrl}${path}`,
      {
        ...options,
        timeoutMs: this.runtimeConfig.timeoutMs,
        headers: {
          Authorization: `Discogs token=${this.runtimeConfig.apiKey}`,
          ...options?.headers
        }
      }
    );
  }

  private get runtimeConfig() {
    return createProviderConfig(
      this.name,
      this.configService.get<string>("DISCOGS_BASE_URL") ??
        "https://api.discogs.com",
      {
        apiKey: this.configService.get<string>("DISCOGS_TOKEN"),
        requiresApiKey: true
      }
    );
  }
}
