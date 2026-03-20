import type {
  AlbumMediaDetails,
  BookMediaDetails,
  GameMediaDetails,
  MediaRecord,
  MovieMediaDetails,
  TvMediaDetails
} from "./media.js";

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
