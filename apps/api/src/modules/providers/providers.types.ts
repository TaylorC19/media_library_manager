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

export type ProviderCacheOperation = "search" | "detail" | "barcode";

export type ProviderErrorCode =
  | "timeout"
  | "network"
  | "rate_limited"
  | "not_found"
  | "invalid_response"
  | "unauthorized"
  | "forbidden"
  | "configuration"
  | "unsupported"
  | "unavailable"
  | "upstream"
  | "unknown";

export interface ProviderCachePolicy {
  operation: ProviderCacheOperation;
  ttlMs: number;
  negativeTtlMs: number;
}

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
  code: ProviderErrorCode;
  message: string;
  statusCode?: number;
  retryable: boolean;
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
