import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import {
  BaseMediaProvider,
  createProviderConfig,
  mapOpenLibraryDetails,
  mapOpenLibrarySearchResult,
  normalizeBarcode,
  normalizeMediaRecord,
  normalizeSearchResults,
  type OpenLibraryBookData,
  type OpenLibrarySearchDoc,
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
import {
  createInvalidProviderResponseError,
  toProviderError
} from "../errors/provider-error.utils";
import { ProviderCacheService } from "../provider-cache.service";
import { ProviderHttpService } from "../http/provider-http.service";

interface OpenLibrarySearchResponse {
  docs?: OpenLibrarySearchDoc[];
}

type OpenLibraryBooksApiResponse = Record<string, OpenLibraryBookData>;

@Injectable()
export class OpenLibraryProvider extends BaseMediaProvider {
  readonly name: ProviderName = "openlibrary";
  readonly capabilities: ProviderCapabilities = {
    mediaTypes: ["book"],
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
    if (!this.enabled || params.mediaType !== "book") {
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
        operation: "search",
        loader: async () => {
          const response = await this.providerHttpService.getJson<OpenLibrarySearchResponse>(
            `${this.runtimeConfig.baseUrl}/search.json`,
            {
              query: {
                q: params.query,
                limit: params.limit ?? 10
              },
              timeoutMs: this.runtimeConfig.timeoutMs
            }
          );

          if (response.docs !== undefined && !Array.isArray(response.docs)) {
            throw createInvalidProviderResponseError(
              this.name,
              "searchByText"
            );
          }

          return normalizeSearchResults(
            (response.docs ?? [])
              .map(mapOpenLibrarySearchResult)
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
    if (!this.enabled || params.mediaType !== "book") {
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
        operation: "detail",
        loader: async () => {
          try {
            const response =
              await this.providerHttpService.getJson<OpenLibraryBooksApiResponse>(
                `${this.runtimeConfig.baseUrl}/api/books`,
                {
                  query: {
                    bibkeys: `OLID:${params.providerId}`,
                    format: "json",
                    jscmd: "data"
                  },
                  timeoutMs: this.runtimeConfig.timeoutMs
                }
              );

            if (!isRecord(response)) {
              throw createInvalidProviderResponseError(this.name, "getDetails");
            }

            const book = response[`OLID:${params.providerId}`];
            if (!book) {
              return null;
            }

            return normalizeMediaRecord(
              mapOpenLibraryDetails(params.providerId, book)
            );
          } catch (error) {
            const providerError = toProviderError(this.name, "getDetails", error);
            if (providerError.code === "not_found") {
              return null;
            }

            throw providerError;
          }
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

    if (params.mediaType !== undefined && params.mediaType !== "book") {
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
        operation: "barcode",
        loader: async () => {
          const response = await this.providerHttpService.getJson<OpenLibrarySearchResponse>(
            `${this.runtimeConfig.baseUrl}/search.json`,
            {
              query: {
                isbn: normalizedBarcode,
                limit: params.limit ?? 10
              },
              timeoutMs: this.runtimeConfig.timeoutMs
            }
          );

          if (response.docs !== undefined && !Array.isArray(response.docs)) {
            throw createInvalidProviderResponseError(
              this.name,
              "searchByBarcode"
            );
          }

          return normalizeSearchResults(
            (response.docs ?? [])
              .map(mapOpenLibrarySearchResult)
              .filter((result): result is NormalizedSearchResult => result !== null)
          );
        }
      });
    } catch (error) {
      throw toProviderError(this.name, "searchByBarcode", error);
    }
  }

  private get runtimeConfig() {
    return createProviderConfig(
      this.name,
      this.configService.get<string>("OPENLIBRARY_BASE_URL") ??
        "https://openlibrary.org"
    );
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
