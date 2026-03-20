import type {
  MediaType,
  NormalizedMediaRecordInput,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";

export type ProviderOperation =
  | "searchByText"
  | "getDetails"
  | "searchByBarcode";

export interface ProviderSearchTextRequest {
  query: string;
  mediaType: MediaType;
  limit?: number;
  providers?: ProviderName[];
}

export interface ProviderBarcodeSearchRequest {
  barcode: string;
  mediaType?: MediaType;
  limit?: number;
  providers?: ProviderName[];
}

export interface ProviderDetailsRequest {
  provider: ProviderName;
  providerId: string;
  mediaType: MediaType;
}

export interface ProviderFailure {
  provider: ProviderName;
  operation: ProviderOperation;
  message: string;
}

export interface ProviderSearchResponse {
  results: NormalizedSearchResult[];
  failures: ProviderFailure[];
}

export interface ProviderBarcodeSearchResponse {
  results: NormalizedSearchResult[];
  failures: ProviderFailure[];
}

export interface ProviderDetailsResponse {
  record: NormalizedMediaRecordInput | null;
  failure?: ProviderFailure;
}
