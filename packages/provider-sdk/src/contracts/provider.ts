import type {
  MediaType,
  NormalizedMediaRecordInput,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";
import {
  type ProviderCapabilities,
  providerSupportsMediaType
} from "./provider-capabilities.js";
import type {
  ProviderGetDetailsParams,
  ProviderSearchByBarcodeParams,
  ProviderSearchByTextParams
} from "./provider-params.js";

export interface MediaProvider {
  readonly name: ProviderName;
  readonly enabled: boolean;
  readonly capabilities: ProviderCapabilities;

  searchByText(
    params: ProviderSearchByTextParams
  ): Promise<NormalizedSearchResult[]>;

  getDetails(
    params: ProviderGetDetailsParams
  ): Promise<NormalizedMediaRecordInput | null>;

  searchByBarcode?(
    params: ProviderSearchByBarcodeParams
  ): Promise<NormalizedSearchResult[]>;

  supportsMediaType(mediaType: MediaType): boolean;
}

export abstract class BaseMediaProvider implements MediaProvider {
  abstract readonly name: ProviderName;
  abstract readonly enabled: boolean;
  abstract readonly capabilities: ProviderCapabilities;

  abstract searchByText(
    params: ProviderSearchByTextParams
  ): Promise<NormalizedSearchResult[]>;

  abstract getDetails(
    params: ProviderGetDetailsParams
  ): Promise<NormalizedMediaRecordInput | null>;

  searchByBarcode(
    params: ProviderSearchByBarcodeParams
  ): Promise<NormalizedSearchResult[]> {
    void params;
    return Promise.resolve([]);
  }

  supportsMediaType(mediaType: MediaType): boolean {
    return providerSupportsMediaType(this.capabilities, mediaType);
  }
}
