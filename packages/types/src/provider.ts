import type { MediaType } from "./library.js";
import type {
  AlbumMediaDetails,
  BookMediaDetails,
  ExternalRatings,
  GameMediaDetails,
  MovieMediaDetails,
  ProviderRefs,
  TvMediaDetails
} from "./media.js";

export const providerNames = [
  "tmdb",
  "musicbrainz",
  "discogs",
  "openlibrary",
  "rawg"
] as const;

export type ProviderName = (typeof providerNames)[number];

export const providerMediaTypes = {
  tmdb: ["movie", "tv"],
  musicbrainz: ["album"],
  discogs: ["album"],
  openlibrary: ["book"],
  rawg: ["game"]
} as const satisfies Record<ProviderName, readonly MediaType[]>;

export type ProviderMediaTypeMap = {
  [Provider in ProviderName]: (typeof providerMediaTypes)[Provider][number];
};

export type ProviderMediaType<Provider extends ProviderName = ProviderName> =
  ProviderMediaTypeMap[Provider];

export interface NormalizedSearchResult {
  provider: ProviderName;
  providerId: string;
  mediaType: MediaType;
  title: string;
  subtitle?: string | null;
  year?: number | null;
  imageUrl?: string | null;
  summary?: string | null;
  creatorLine?: string | null;
  barcodeCandidates?: string[];
  confidence?: number | null;
}

interface NormalizedMediaRecordInputBase {
  mediaType: MediaType;
  title: string;
  sortTitle?: string | null;
  releaseDate?: string | null;
  year?: number | null;
  imageUrl?: string | null;
  summary?: string | null;
  providerRefs: ProviderRefs;
  externalRatings?: ExternalRatings;
  barcodeCandidates?: string[];
}

export interface NormalizedMovieMediaRecordInput
  extends NormalizedMediaRecordInputBase {
  mediaType: "movie";
  details: MovieMediaDetails;
}

export interface NormalizedTvMediaRecordInput
  extends NormalizedMediaRecordInputBase {
  mediaType: "tv";
  details: TvMediaDetails;
}

export interface NormalizedAlbumMediaRecordInput
  extends NormalizedMediaRecordInputBase {
  mediaType: "album";
  details: AlbumMediaDetails;
}

export interface NormalizedBookMediaRecordInput
  extends NormalizedMediaRecordInputBase {
  mediaType: "book";
  details: BookMediaDetails;
}

export interface NormalizedGameMediaRecordInput
  extends NormalizedMediaRecordInputBase {
  mediaType: "game";
  details: GameMediaDetails;
}

export type NormalizedMediaRecordInput =
  | NormalizedMovieMediaRecordInput
  | NormalizedTvMediaRecordInput
  | NormalizedAlbumMediaRecordInput
  | NormalizedBookMediaRecordInput
  | NormalizedGameMediaRecordInput;
