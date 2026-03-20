import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import {
  BaseMediaProvider,
  buildTmdbSubtitle,
  createProviderConfig,
  mapTmdbDetails,
  mapTmdbSearchResult,
  normalizeMediaRecord,
  normalizeSearchResults,
  type ProviderCapabilities,
  type ProviderGetDetailsParams,
  type ProviderSearchByTextParams,
  type TmdbDetailsResponse,
  type TmdbSearchResultItem
} from "@media-library/provider-sdk";
import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";
import { toProviderError } from "../errors/provider-error.utils";
import { ProviderCacheService } from "../provider-cache.service";
import { ProviderHttpService } from "../http/provider-http.service";

interface TmdbSearchResponse {
  results: TmdbSearchResultItem[];
}

@Injectable()
export class TmdbProvider extends BaseMediaProvider {
  readonly name: ProviderName = "tmdb";
  readonly capabilities: ProviderCapabilities = {
    mediaTypes: ["movie", "tv"],
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
    const mediaType = params.mediaType;

    if (!this.enabled || (mediaType !== "movie" && mediaType !== "tv")) {
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
          const response = await this.providerHttpService.getJson<TmdbSearchResponse>(
            `${this.runtimeConfig.baseUrl}/search/${mediaType}`,
            {
              query: {
                api_key: this.runtimeConfig.apiKey,
                query: params.query,
                page: 1
              },
              timeoutMs: this.runtimeConfig.timeoutMs
            }
          );

          return normalizeSearchResults(
            response.results.slice(0, params.limit ?? 10).map((item) => ({
              ...mapTmdbSearchResult(item, mediaType),
              subtitle: buildTmdbSubtitle(item, mediaType)
            }))
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
    const mediaType = params.mediaType;

    if (!this.enabled || (mediaType !== "movie" && mediaType !== "tv")) {
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
          const response = await this.providerHttpService.getJson<TmdbDetailsResponse>(
            `${this.runtimeConfig.baseUrl}/${mediaType}/${params.providerId}`,
            {
              query: {
                api_key: this.runtimeConfig.apiKey,
                append_to_response: "credits"
              },
              timeoutMs: this.runtimeConfig.timeoutMs
            }
          );

          return normalizeMediaRecord(
            mapTmdbDetails(response, mediaType)
          );
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "getDetails", error);
    }
  }

  private get runtimeConfig() {
    return createProviderConfig(
      this.name,
      this.configService.get<string>("TMDB_BASE_URL") ??
        "https://api.themoviedb.org/3",
      {
        apiKey: this.configService.get<string>("TMDB_API_KEY"),
        requiresApiKey: true
      }
    );
  }
}
