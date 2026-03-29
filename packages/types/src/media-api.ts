import type {
  AlbumMediaDetails,
  BookMediaDetails,
  GameMediaDetails,
  MediaRecord,
  MovieMediaDetails,
  TvMediaDetails
} from "./media.js";
import type { LibraryBucket } from "./library.js";
import type { LibraryEntryResponse } from "./library-api.js";
import type {
  NormalizedSearchResult,
  ProviderName
} from "./provider.js";
import type { MediaType, PhysicalFormat } from "./library.js";

export interface ManualMediaRecordBaseInput {
  title: string;
  year?: number | null;
  releaseDate?: string | null;
  imageUrl?: string | null;
  summary?: string | null;
  barcodeCandidates?: string[];
}

export interface CreateManualMovieMediaRecordRequest
  extends ManualMediaRecordBaseInput {
  mediaType: "movie";
  details?: MovieMediaDetails;
}

export interface CreateManualTvMediaRecordRequest
  extends ManualMediaRecordBaseInput {
  mediaType: "tv";
  details?: TvMediaDetails;
}

export interface CreateManualAlbumMediaRecordRequest
  extends ManualMediaRecordBaseInput {
  mediaType: "album";
  details: AlbumMediaDetails;
}

export interface CreateManualBookMediaRecordRequest
  extends ManualMediaRecordBaseInput {
  mediaType: "book";
  details: BookMediaDetails;
}

export interface CreateManualGameMediaRecordRequest
  extends ManualMediaRecordBaseInput {
  mediaType: "game";
  details?: GameMediaDetails;
}

export type CreateManualMediaRecordRequest =
  | CreateManualMovieMediaRecordRequest
  | CreateManualTvMediaRecordRequest
  | CreateManualAlbumMediaRecordRequest
  | CreateManualBookMediaRecordRequest
  | CreateManualGameMediaRecordRequest;

export interface ManualMediaRecordResponse {
  mediaRecord: MediaRecord;
}

interface ImportMediaRecordRequestBase {
  mode: "provider_ref" | "search_result";
  entry?: ImportMediaEntryInput;
}

export interface ImportMediaEntryInput {
  bucket: LibraryBucket;
  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;
  notes?: string | null;
  tags?: string[];
}

export interface ImportMediaRecordFromProviderRefRequest
  extends ImportMediaRecordRequestBase {
  mode: "provider_ref";
  provider: ProviderName;
  providerId: string;
  mediaType: MediaType;
}

export interface ImportMediaRecordFromSearchResultRequest
  extends ImportMediaRecordRequestBase {
  mode: "search_result";
  result: NormalizedSearchResult;
}

export type ImportMediaRecordRequest =
  | ImportMediaRecordFromProviderRefRequest
  | ImportMediaRecordFromSearchResultRequest;

export interface ImportMediaRecordResponse {
  mediaRecord: MediaRecord;
  libraryEntry?: LibraryEntryResponse;
  wasExistingMediaRecord: boolean;
  wasExistingLibraryEntry?: boolean;
}

export interface GetMediaRecordResponse {
  mediaRecord: MediaRecord;
}

export const refreshMediaRecordUnavailableReasons = [
  "provider_ref_unavailable",
  "provider_record_unavailable"
] as const;

export type RefreshMediaRecordUnavailableReason =
  (typeof refreshMediaRecordUnavailableReasons)[number];

export interface RefreshMediaRecordResponse {
  mediaRecord: MediaRecord;
  wasRefreshed: boolean;
  refreshedFromProvider?: ProviderName;
  unavailableReason?: RefreshMediaRecordUnavailableReason;
}
