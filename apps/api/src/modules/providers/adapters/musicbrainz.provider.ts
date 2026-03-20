import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import {
  BaseMediaProvider,
  createProviderConfig,
  mapMusicBrainzDetails,
  mapMusicBrainzSearchResult,
  normalizeBarcode,
  normalizeMediaRecord,
  normalizeSearchResults,
  type MusicBrainzRelease,
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
import { MusicBrainzThrottleService } from "../http/musicbrainz-throttle.service";
import { ProviderHttpService } from "../http/provider-http.service";

interface MusicBrainzReleaseSearchResponse {
  releases?: MusicBrainzRelease[];
}

@Injectable()
export class MusicBrainzProvider extends BaseMediaProvider {
  readonly name: ProviderName = "musicbrainz";
  readonly capabilities: ProviderCapabilities = {
    mediaTypes: ["album"],
    supportsTextSearch: true,
    supportsDetailLookup: true,
    supportsBarcodeSearch: true
  };

  constructor(
    private readonly configService: ConfigService,
    private readonly providerHttpService: ProviderHttpService,
    private readonly providerCacheService: ProviderCacheService,
    private readonly musicBrainzThrottleService: MusicBrainzThrottleService
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
          const response = await this.request<MusicBrainzReleaseSearchResponse>(
            "/release",
            {
              query: {
                query: params.query,
                fmt: "json",
                limit: params.limit ?? 10
              }
            }
          );

          return normalizeSearchResults(
            (response.releases ?? []).map(mapMusicBrainzSearchResult)
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
          const response = await this.request<MusicBrainzRelease>(
            `/release/${params.providerId}`,
            {
              query: {
                fmt: "json",
                inc: "artists+labels+recordings"
              }
            }
          );

          return normalizeMediaRecord(mapMusicBrainzDetails(response));
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
          const response = await this.request<MusicBrainzReleaseSearchResponse>(
            "/release",
            {
              query: {
                query: `barcode:${normalizedBarcode}`,
                fmt: "json",
                limit: params.limit ?? 10
              }
            }
          );

          return normalizeSearchResults(
            (response.releases ?? []).map(mapMusicBrainzSearchResult)
          );
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "searchByBarcode", error);
    }
  }

  private request<Response>(
    path: string,
    options: Parameters<ProviderHttpService["getJson"]>[1]
  ): Promise<Response> {
    return this.musicBrainzThrottleService.schedule(() =>
      this.providerHttpService.getJson<Response>(
        `${this.runtimeConfig.baseUrl}${path}`,
        {
          ...options,
          timeoutMs: this.runtimeConfig.timeoutMs,
          headers: {
            "user-agent": this.runtimeConfig.userAgent ?? "",
            ...options?.headers
          }
        }
      )
    );
  }

  private get runtimeConfig() {
    const userAgent = this.configService.get<string>("MUSICBRAINZ_USER_AGENT");
    return createProviderConfig(
      this.name,
      this.configService.get<string>("MUSICBRAINZ_BASE_URL") ??
        "https://musicbrainz.org/ws/2",
      {
        userAgent,
        enabled:
          Boolean(
            this.configService.get<string>("MUSICBRAINZ_BASE_URL") ??
              "https://musicbrainz.org/ws/2"
          ) && Boolean(userAgent?.trim())
      }
    );
  }
}
