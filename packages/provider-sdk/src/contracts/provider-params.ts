import type { MediaType } from "@media-library/types";

export interface ProviderSearchByTextParams {
  query: string;
  mediaType: MediaType;
  limit?: number;
}

export interface ProviderGetDetailsParams {
  providerId: string;
  mediaType: MediaType;
}

export interface ProviderSearchByBarcodeParams {
  barcode: string;
  mediaType?: MediaType;
  limit?: number;
}
