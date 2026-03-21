import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import {
  BaseMediaProvider,
  createProviderConfig,
  mapRawgDetails,
  mapRawgSearchResult,
  normalizeMediaRecord,
  normalizeSearchResults,
  type ProviderCapabilities,
  type ProviderGetDetailsParams,
  type ProviderSearchByTextParams,
  type RawgDetailsResponse,
  type RawgSearchResultItem
} from "@media-library/provider-sdk";
import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";
import { toProviderError } from "../errors/provider-error.utils";
import { ProviderCacheService } from "../provider-cache.service";
import { ProviderHttpService } from "../http/provider-http.service";

interface RawgSearchResponse {
  results?: RawgSearchResultItem[];
}

@Injectable()
export class RawgProvider extends BaseMediaProvider {
  readonly name: ProviderName = "rawg";
  readonly capabilities: ProviderCapabilities = {
    mediaTypes: ["game"],
    supportsTextSearch: true,
    supportsDetailLookup: true,
    supportsBarcodeSearch: false
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
    if (!this.enabled || params.mediaType !== "game") {
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
        negativeTtlMs: this.providerCacheService.getNegativeSearchTtlMs(),
        loader: async () => {
          const response = await this.providerHttpService.getJson<RawgSearchResponse>(
            `${this.runtimeConfig.baseUrl}/games`,
            {
              query: {
                key: this.runtimeConfig.apiKey,
                search: params.query,
                page_size: params.limit ?? 10
              },
              timeoutMs: this.runtimeConfig.timeoutMs
            }
          );

          return normalizeSearchResults(
            (response.results ?? []).map(mapRawgSearchResult)
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
    if (!this.enabled || params.mediaType !== "game") {
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
        negativeTtlMs: this.providerCacheService.getNegativeDetailsTtlMs(),
        loader: async () => {
          const response = await this.providerHttpService.getJson<RawgDetailsResponse>(
            `${this.runtimeConfig.baseUrl}/games/${params.providerId}`,
            {
              query: {
                key: this.runtimeConfig.apiKey
              },
              timeoutMs: this.runtimeConfig.timeoutMs
            }
          );

          return normalizeMediaRecord(mapRawgDetails(response));
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "getDetails", error);
    }
  }

  private get runtimeConfig() {
    return createProviderConfig(
      this.name,
      this.configService.get<string>("RAWG_BASE_URL") ??
        "https://api.rawg.io/api",
      {
        apiKey: this.configService.get<string>("RAWG_API_KEY"),
        requiresApiKey: true
      }
    );
  }
}
