import type {
  MediaType,
  NormalizedMediaRecordInput,
  NormalizedSearchResult,
  ProviderName
} from "@media-library/types";

export interface ProviderSearchParams {
  query: string;
  mediaType: MediaType;
}

export interface ProviderImportParams {
  providerId: string;
  mediaType: MediaType;
}

export interface MediaProvider {
  readonly name: ProviderName;

  search(params: ProviderSearchParams): Promise<NormalizedSearchResult[]>;

  getMediaRecord(
    params: ProviderImportParams
  ): Promise<NormalizedMediaRecordInput | null>;
}
